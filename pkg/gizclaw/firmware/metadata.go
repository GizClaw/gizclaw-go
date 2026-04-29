package firmware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

var depotMetadataRoot = kv.Key{"by-name"}

func (s *Server) metadataStore() (kv.Store, error) {
	if s.MetadataStore == nil {
		return nil, fmt.Errorf("firmware: metadata store not configured")
	}
	return s.MetadataStore, nil
}

func depotMetadataKey(name string) kv.Key {
	parts := strings.Split(name, "/")
	key := make(kv.Key, 0, len(depotMetadataRoot)+len(parts))
	key = append(key, depotMetadataRoot...)
	key = append(key, parts...)
	return key
}

func depotNameFromMetadataKey(key kv.Key) (string, bool) {
	if len(key) <= len(depotMetadataRoot) {
		return "", false
	}
	for i, segment := range depotMetadataRoot {
		if key[i] != segment {
			return "", false
		}
	}
	return strings.Join(key[len(depotMetadataRoot):], "/"), true
}

func (s *Server) listDepotMetadata(ctx context.Context) ([]apitypes.Depot, error) {
	store, err := s.metadataStore()
	if err != nil {
		return nil, err
	}
	var out []apitypes.Depot
	for entry, err := range store.List(ctx, depotMetadataRoot) {
		if err != nil {
			return nil, err
		}
		depot, err := decodeDepotMetadata(entry.Value)
		if err != nil {
			return nil, err
		}
		if depot.Name == "" {
			name, ok := depotNameFromMetadataKey(entry.Key)
			if !ok {
				return nil, fmt.Errorf("firmware: invalid depot metadata key %s", entry.Key.String())
			}
			depot.Name = name
		}
		out = append(out, depot)
	}
	return out, nil
}

func (s *Server) getDepotMetadata(ctx context.Context, name string) (apitypes.Depot, error) {
	if err := validateDepotName(name); err != nil {
		return apitypes.Depot{}, err
	}
	store, err := s.metadataStore()
	if err != nil {
		return apitypes.Depot{}, err
	}
	data, err := store.Get(ctx, depotMetadataKey(name))
	if err != nil {
		if errors.Is(err, kv.ErrNotFound) {
			return apitypes.Depot{}, errDepotNotFound
		}
		return apitypes.Depot{}, err
	}
	depot, err := decodeDepotMetadata(data)
	if err != nil {
		return apitypes.Depot{}, err
	}
	if depot.Name == "" {
		depot.Name = name
	}
	if depot.Name != name {
		return apitypes.Depot{}, fmt.Errorf("firmware: depot metadata name mismatch %q != %q", depot.Name, name)
	}
	if err := validateDepotMetadata(depot); err != nil {
		return apitypes.Depot{}, err
	}
	return depot, nil
}

func (s *Server) writeDepotMetadata(ctx context.Context, depot apitypes.Depot) error {
	if err := validateDepotName(depot.Name); err != nil {
		return err
	}
	depot = normalizeDepotMetadata(depot)
	if err := validateDepotMetadata(depot); err != nil {
		return err
	}
	store, err := s.metadataStore()
	if err != nil {
		return err
	}
	data, err := json.Marshal(depot)
	if err != nil {
		return fmt.Errorf("firmware: encode depot %s: %w", depot.Name, err)
	}
	return store.Set(ctx, depotMetadataKey(depot.Name), data)
}

func decodeDepotMetadata(data []byte) (apitypes.Depot, error) {
	var depot apitypes.Depot
	if err := json.Unmarshal(data, &depot); err != nil {
		return apitypes.Depot{}, fmt.Errorf("firmware: decode depot metadata: %w", err)
	}
	return normalizeDepotMetadata(depot), nil
}

func normalizeDepotMetadata(depot apitypes.Depot) apitypes.Depot {
	depot.Info = normalizeDepotInfo(depot.Info)
	depot.Rollback = normalizeDepotRelease(depot.Rollback)
	depot.Stable = normalizeDepotRelease(depot.Stable)
	depot.Beta = normalizeDepotRelease(depot.Beta)
	depot.Testing = normalizeDepotRelease(depot.Testing)
	return depot
}

func validateDepotMetadata(depot apitypes.Depot) error {
	if err := validateDepotName(depot.Name); err != nil {
		return err
	}
	if err := validateDepotInfo(depot.Info); err != nil {
		return err
	}
	for _, item := range []struct {
		channel Channel
		release apitypes.DepotRelease
	}{
		{Rollback, depot.Rollback},
		{Stable, depot.Stable},
		{Beta, depot.Beta},
		{Testing, depot.Testing},
	} {
		release := item.release
		if release.FirmwareSemver == "" {
			continue
		}
		if err := validateRelease(release); err != nil {
			return err
		}
		if releaseChannel(release) != item.channel {
			return fmt.Errorf("firmware: depot %s %s release channel mismatch", depot.Name, item.channel)
		}
	}
	return validateVersionOrder(depot)
}

func releaseWithChannel(release apitypes.DepotRelease, channel Channel) apitypes.DepotRelease {
	if release.FirmwareSemver == "" {
		return apitypes.DepotRelease{}
	}
	release.Channel = stringPtr(string(channel))
	return normalizeDepotRelease(release)
}
