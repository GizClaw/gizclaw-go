package firmware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/giztoy/giztoy-go/pkg/httputil"
)

// Manager bundles the firmware depot store, scanner, and derived services.
type Manager struct {
	store      *Store
	scanner    *Scanner
	uploader   *Uploader
	switcher   *Switcher
	ota        *OTAService
	pathPrefix string
	adminMux   *http.ServeMux
}

// NewManager constructs a Manager for the given depot filesystem store.
// pathPrefix is the URL mount point for admin endpoints (e.g. "/firmwares").
func NewManager(store *Store, pathPrefix string) *Manager {
	sc := NewScanner(store)
	m := &Manager{
		store:      store,
		scanner:    sc,
		uploader:   NewUploader(store, sc),
		switcher:   NewSwitcher(store, sc),
		ota:        NewOTAService(store, sc),
		pathPrefix: pathPrefix,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+pathPrefix, m.handleListDepots)
	mux.HandleFunc("GET "+pathPrefix+"/{depot}", m.handleGetDepot)
	mux.HandleFunc("PUT "+pathPrefix+"/{depot}", m.handlePutDepot)
	mux.HandleFunc("GET "+pathPrefix+"/{depot}/{channel}", m.handleGetChannel)
	mux.HandleFunc("PUT "+pathPrefix+"/{depot}/{channel}", m.handlePutChannel)
	m.adminMux = mux
	return m
}

// ---------------------------------------------------------------------------
// Public API (delegate methods)
// ---------------------------------------------------------------------------

// PathPrefix returns the URL path prefix this manager's routes are mounted at.
func (m *Manager) PathPrefix() string {
	return m.pathPrefix
}

// AdminHandler returns the pre-built http.Handler for firmware admin endpoints.
func (m *Manager) AdminHandler() http.Handler {
	return m.adminMux
}

// ResolveOTA returns the OTA summary for the given depot and channel.
func (m *Manager) ResolveOTA(depot string, channel Channel) (OTASummary, error) {
	return m.ota.Resolve(depot, channel)
}

// ResolveOTAFile returns the full path and file metadata for a firmware file.
func (m *Manager) ResolveOTAFile(depot string, channel Channel, path string) (string, DepotFile, error) {
	return m.ota.ResolveFile(depot, channel, path)
}

// PutInfo creates or updates the depot info (file manifest).
func (m *Manager) PutInfo(depot string, info DepotInfo) error {
	return m.uploader.PutInfo(depot, info)
}

// UploadTar uploads a firmware release tarball for the given depot and channel.
func (m *Manager) UploadTar(depot string, channel Channel, r io.Reader) (DepotRelease, error) {
	return m.uploader.UploadTar(depot, channel, r)
}

// ManifestPath returns the filesystem path of the manifest for a depot channel.
func (m *Manager) ManifestPath(depot string, channel Channel) string {
	return m.store.ManifestPath(depot, channel)
}

// ---------------------------------------------------------------------------
// HTTP admin handler
// ---------------------------------------------------------------------------

var (
	httpWriteJSON  = httputil.WriteJSON
	httpWriteError = httputil.WriteError
)

func (m *Manager) handleListDepots(w http.ResponseWriter, r *http.Request) {
	depots, err := m.scanner.Scan()
	if err != nil {
		httpWriteError(w, http.StatusInternalServerError, "DIRECTORY_SCAN_FAILED", err.Error())
		return
	}
	httpWriteJSON(w, http.StatusOK, map[string]any{"items": depots})
}

func (m *Manager) handleGetDepot(w http.ResponseWriter, r *http.Request) {
	depot := r.PathValue("depot")
	snapshot, err := m.scanner.ScanDepot(depot)
	if err != nil {
		httpWriteError(w, http.StatusNotFound, "DEPOT_NOT_FOUND", err.Error())
		return
	}
	httpWriteJSON(w, http.StatusOK, snapshot)
}

// handlePutDepot dispatches depot PUT requests: plain depot info update,
// or :rollback / :release actions encoded as a suffix in the depot segment.
func (m *Manager) handlePutDepot(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("depot")
	switch {
	case strings.HasSuffix(raw, ":rollback"):
		depot := strings.TrimSuffix(raw, ":rollback")
		snapshot, err := m.switcher.Rollback(depot)
		if err != nil {
			writeSwitchError(w, "rollback", err)
			return
		}
		httpWriteJSON(w, http.StatusOK, snapshot)
	case strings.HasSuffix(raw, ":release"):
		depot := strings.TrimSuffix(raw, ":release")
		snapshot, err := m.switcher.Release(depot)
		if err != nil {
			writeSwitchError(w, "release", err)
			return
		}
		httpWriteJSON(w, http.StatusOK, snapshot)
	default:
		m.handlePutDepotInfo(w, r, raw)
	}
}

func (m *Manager) handlePutDepotInfo(w http.ResponseWriter, r *http.Request, depot string) {
	var info DepotInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		httpWriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if err := m.uploader.PutInfo(depot, info); err != nil {
		httpWriteError(w, http.StatusConflict, "INFO_FILES_MISMATCH", err.Error())
		return
	}
	snapshot, err := m.scanner.ScanDepot(depot)
	if err != nil {
		httpWriteError(w, http.StatusInternalServerError, "DIRECTORY_SCAN_FAILED", err.Error())
		return
	}
	httpWriteJSON(w, http.StatusOK, snapshot)
}

func writeSwitchError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, ErrDepotNotFound):
		httpWriteError(w, http.StatusNotFound, "DEPOT_NOT_FOUND", err.Error())
	case errors.Is(err, ErrChannelNotFound):
		if op == "rollback" {
			httpWriteError(w, http.StatusConflict, "ROLLBACK_NOT_AVAILABLE", err.Error())
			return
		}
		httpWriteError(w, http.StatusConflict, "RELEASE_NOT_READY", err.Error())
	default:
		httpWriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func (m *Manager) handleGetChannel(w http.ResponseWriter, r *http.Request) {
	depot := r.PathValue("depot")
	channel := Channel(r.PathValue("channel"))
	snapshot, err := m.scanner.ScanDepot(depot)
	if err != nil {
		httpWriteError(w, http.StatusNotFound, "DEPOT_NOT_FOUND", err.Error())
		return
	}
	release, ok := snapshot.Release(channel)
	if !ok {
		httpWriteError(w, http.StatusNotFound, "CHANNEL_NOT_FOUND", "channel not found")
		return
	}
	httpWriteJSON(w, http.StatusOK, release)
}

func (m *Manager) handlePutChannel(w http.ResponseWriter, r *http.Request) {
	depot := r.PathValue("depot")
	channel := Channel(r.PathValue("channel"))
	release, err := m.uploader.UploadTar(depot, channel, r.Body)
	if err != nil {
		httpWriteError(w, http.StatusConflict, "MANIFEST_INVALID", err.Error())
		return
	}
	httpWriteJSON(w, http.StatusOK, release)
}
