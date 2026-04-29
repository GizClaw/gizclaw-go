package firmware

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/store/depotstore"
)

func (s *Server) uploadTar(ctx context.Context, depot string, channel Channel, r io.Reader) (apitypes.DepotRelease, error) {
	if !isValidChannel(channel) {
		return apitypes.DepotRelease{}, fmt.Errorf("firmware: invalid channel %q", channel)
	}
	unlock := s.lockDepot(depot)
	defer unlock()
	if err := s.ensureDepot(depot); err != nil {
		return apitypes.DepotRelease{}, err
	}
	tmpPath := s.tempPath(depot, string(channel))
	_ = s.store().RemoveAll(tmpPath)
	if err := s.store().MkdirAll(tmpPath); err != nil {
		return apitypes.DepotRelease{}, err
	}
	defer s.store().RemoveAll(tmpPath)

	release, err := extractTar(s.store(), tmpPath, channel, r)
	if err != nil {
		return apitypes.DepotRelease{}, err
	}
	snapshot, err := s.scanDepot(ctx, depot)
	if err != nil {
		if !errors.Is(err, errDepotNotFound) {
			return apitypes.DepotRelease{}, err
		}
		snapshot = apitypes.Depot{Name: depot}
	} else if len(infoFiles(snapshot.Info)) > 0 {
		if !sameInfoFiles(snapshot.Info, release) {
			return apitypes.DepotRelease{}, fmt.Errorf("firmware: info files mismatch")
		}
	}

	targetPath := s.channelPath(depot, string(channel))
	swapPath := targetPath + ".old"
	_ = s.store().RemoveAll(swapPath)
	hadPrevious := false
	if _, err := s.store().Stat(targetPath); err == nil {
		hadPrevious = true
		if err := s.store().Rename(targetPath, swapPath); err != nil {
			return apitypes.DepotRelease{}, err
		}
	}
	if err := s.store().Rename(tmpPath, targetPath); err != nil {
		if hadPrevious {
			_ = s.store().Rename(swapPath, targetPath)
		}
		return apitypes.DepotRelease{}, err
	}

	setDepotRelease(&snapshot, channel, normalizeDepotRelease(release))
	if err := s.writeDepotMetadata(ctx, snapshot); err != nil {
		_ = s.store().RemoveAll(targetPath)
		if hadPrevious {
			_ = s.store().Rename(swapPath, targetPath)
		}
		return apitypes.DepotRelease{}, err
	}
	_ = s.store().RemoveAll(swapPath)
	return normalizeDepotRelease(release), nil
}

func extractTar(store depotstore.Store, dst string, wantChannel Channel, r io.Reader) (apitypes.DepotRelease, error) {
	tr := tar.NewReader(r)
	seen := make(map[string]struct{})
	var manifest apitypes.DepotRelease
	var manifestLoaded bool
	files := make(map[string][]byte)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apitypes.DepotRelease{}, err
		}
		if hdr.Typeflag != tar.TypeReg {
			return apitypes.DepotRelease{}, fmt.Errorf("firmware: illegal tar entry %s", hdr.Name)
		}
		if err := validateRelativePath(hdr.Name); err != nil {
			return apitypes.DepotRelease{}, err
		}
		if _, ok := seen[hdr.Name]; ok {
			return apitypes.DepotRelease{}, fmt.Errorf("firmware: duplicate tar entry %s", hdr.Name)
		}
		seen[hdr.Name] = struct{}{}
		data, err := io.ReadAll(tr)
		if err != nil {
			return apitypes.DepotRelease{}, err
		}
		if hdr.Name == "manifest.json" {
			manifest, err = parseManifest(data)
			if err != nil {
				return apitypes.DepotRelease{}, err
			}
			if releaseChannel(manifest) != wantChannel {
				return apitypes.DepotRelease{}, fmt.Errorf("firmware: manifest channel mismatch")
			}
			manifestLoaded = true
			continue
		}
		files[hdr.Name] = data
	}
	if !manifestLoaded {
		return apitypes.DepotRelease{}, fmt.Errorf("firmware: manifest missing")
	}
	for _, file := range releaseFiles(manifest) {
		data, ok := files[file.Path]
		if !ok {
			return apitypes.DepotRelease{}, fmt.Errorf("firmware: missing manifest file %s", file.Path)
		}
		target := path.Join(dst, file.Path)
		if err := store.WriteFile(target, data); err != nil {
			return apitypes.DepotRelease{}, err
		}
		delete(files, file.Path)
	}
	if len(files) > 0 {
		return apitypes.DepotRelease{}, fmt.Errorf("firmware: tar files mismatch")
	}
	if err := writeManifest(store, path.Join(dst, "manifest.json"), manifest); err != nil {
		return apitypes.DepotRelease{}, err
	}
	if err := validateReleaseAgainstFiles(store, dst, manifest); err != nil {
		return apitypes.DepotRelease{}, err
	}
	return manifest, nil
}
