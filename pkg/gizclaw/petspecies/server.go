package petspecies

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
	"github.com/GizClaw/gizclaw-go/pkg/store/objectstore"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

var rootKey = kv.Key{"by-id"}

type Server struct {
	Store  kv.Store
	Assets objectstore.ObjectStore
	Now    func() time.Time
}

func (s *Server) Put(ctx context.Context, id string, spec apitypes.PetSpeciesSpec) (apitypes.PetSpecies, error) {
	store, err := s.store()
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return apitypes.PetSpecies{}, errors.New("pet species id is required")
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return apitypes.PetSpecies{}, errors.New("pet species name is required")
	}
	now := s.now()
	current, err := Get(ctx, store, id)
	if err != nil && !errors.Is(err, kv.ErrNotFound) {
		return apitypes.PetSpecies{}, err
	}
	out := apitypes.PetSpecies{
		Id:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err == nil {
		out.CreatedAt = current.CreatedAt
		out.ZpetPath = current.ZpetPath
		out.ZpetMetadata = current.ZpetMetadata
	}
	if spec.ZpetPath != nil {
		out.ZpetPath = strings.TrimSpace(*spec.ZpetPath)
	}
	if err := Write(ctx, store, out); err != nil {
		return apitypes.PetSpecies{}, err
	}
	return out, nil
}

func (s *Server) Get(ctx context.Context, id string) (apitypes.PetSpecies, error) {
	store, err := s.store()
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	return Get(ctx, store, id)
}

func (s *Server) Delete(ctx context.Context, id string) (apitypes.PetSpecies, error) {
	store, err := s.store()
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	item, err := Get(ctx, store, id)
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	if err := store.Delete(ctx, speciesKey(id)); err != nil {
		return apitypes.PetSpecies{}, err
	}
	if item.ZpetPath != "" && s.Assets != nil {
		if err := s.Assets.Delete(item.ZpetPath); err != nil {
			return apitypes.PetSpecies{}, err
		}
	}
	return item, nil
}

func (s *Server) List(ctx context.Context, cursor string, limit int) ([]apitypes.PetSpecies, bool, *string, error) {
	store, err := s.store()
	if err != nil {
		return nil, false, nil, err
	}
	normalizedCursor, normalizedLimit := normalizeListParams(cursor, limit)
	entries, err := kv.ListAfter(ctx, store, rootKey, cursorAfterKey(normalizedCursor), normalizedLimit+1)
	if err != nil {
		return nil, false, nil, err
	}
	hasNext := len(entries) > normalizedLimit
	if hasNext {
		entries = entries[:normalizedLimit]
	}
	items := make([]apitypes.PetSpecies, 0, len(entries))
	for _, entry := range entries {
		var item apitypes.PetSpecies
		if err := json.Unmarshal(entry.Value, &item); err != nil {
			return nil, false, nil, err
		}
		items = append(items, item)
	}
	var next *string
	if hasNext && len(entries) > 0 {
		v := unescapeStoreSegment(entries[len(entries)-1].Key[len(entries[len(entries)-1].Key)-1])
		next = &v
	}
	return items, hasNext, next, nil
}

func (s *Server) UploadZpet(ctx context.Context, id string, r io.Reader) (apitypes.PetSpecies, error) {
	store, err := s.store()
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	assets, err := s.assets()
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	item, err := Get(ctx, store, id)
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	metadata, err := ParseZpetMetadata(data)
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	if item.ZpetPath == "" {
		item.ZpetPath = item.Id + ".zpet"
	}
	if err := assets.Put(item.ZpetPath, bytes.NewReader(data)); err != nil {
		return apitypes.PetSpecies{}, err
	}
	item.ZpetMetadata = metadata
	item.UpdatedAt = s.now()
	if err := Write(ctx, store, item); err != nil {
		return apitypes.PetSpecies{}, err
	}
	return item, nil
}

func (s *Server) DownloadZpet(ctx context.Context, id string) (io.ReadCloser, error) {
	store, err := s.store()
	if err != nil {
		return nil, err
	}
	assets, err := s.assets()
	if err != nil {
		return nil, err
	}
	item, err := Get(ctx, store, id)
	if err != nil {
		return nil, err
	}
	if item.ZpetPath == "" {
		return nil, fmt.Errorf("pet species %q has no zpet file", id)
	}
	return assets.Get(item.ZpetPath)
}

func Get(ctx context.Context, store kv.Store, id string) (apitypes.PetSpecies, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return apitypes.PetSpecies{}, errors.New("pet species id is required")
	}
	data, err := store.Get(ctx, speciesKey(id))
	if err != nil {
		return apitypes.PetSpecies{}, err
	}
	var item apitypes.PetSpecies
	if err := json.Unmarshal(data, &item); err != nil {
		return apitypes.PetSpecies{}, err
	}
	return item, nil
}

func Write(ctx context.Context, store kv.Store, item apitypes.PetSpecies) error {
	if strings.TrimSpace(item.Id) == "" {
		return errors.New("pet species id is required")
	}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return store.Set(ctx, speciesKey(item.Id), data)
}

func ParseZpetMetadata(data []byte) (apitypes.ZpetMetadata, error) {
	line, err := bufio.NewReader(bytes.NewReader(data)).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return apitypes.ZpetMetadata{}, err
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return apitypes.ZpetMetadata{}, errors.New("zpet metadata line is empty")
	}
	var header struct {
		Magic   string `json:"magic"`
		Version int    `json:"version"`
		ID      string `json:"id"`
		Canvas  []int  `json:"canvas"`
		Format  string `json:"format"`
		Clips   []struct {
			ID string `json:"id"`
		} `json:"clips"`
	}
	if err := json.Unmarshal(line, &header); err != nil {
		return apitypes.ZpetMetadata{}, fmt.Errorf("invalid zpet metadata: %w", err)
	}
	if header.Magic != "zpet" {
		return apitypes.ZpetMetadata{}, fmt.Errorf("invalid zpet magic %q", header.Magic)
	}
	if header.Version == 0 || strings.TrimSpace(header.ID) == "" || len(header.Canvas) != 2 || strings.TrimSpace(header.Format) == "" {
		return apitypes.ZpetMetadata{}, errors.New("invalid zpet metadata: version, id, canvas, and format are required")
	}
	clipIDs := make([]string, 0, len(header.Clips))
	for _, clip := range header.Clips {
		id := strings.TrimSpace(clip.ID)
		if id != "" {
			clipIDs = append(clipIDs, id)
		}
	}
	return apitypes.ZpetMetadata{
		Version:      header.Version,
		SpeciesId:    strings.TrimSpace(header.ID),
		CanvasWidth:  header.Canvas[0],
		CanvasHeight: header.Canvas[1],
		Format:       strings.TrimSpace(header.Format),
		ClipIds:      clipIDs,
	}, nil
}

func (s *Server) store() (kv.Store, error) {
	if s == nil || s.Store == nil {
		return nil, errors.New("pet species service not configured")
	}
	return s.Store, nil
}

func (s *Server) assets() (objectstore.ObjectStore, error) {
	if s == nil || s.Assets == nil {
		return nil, errors.New("pet species asset store not configured")
	}
	return s.Assets, nil
}

func (s *Server) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func speciesKey(id string) kv.Key {
	return append(append(kv.Key{}, rootKey...), escapeStoreSegment(strings.TrimSpace(id)))
}

func normalizeListParams(cursor string, limit int) (string, int) {
	normalizedCursor := escapeStoreSegment(strings.TrimSpace(cursor))
	normalizedLimit := defaultListLimit
	if limit > 0 {
		normalizedLimit = limit
	}
	if normalizedLimit > maxListLimit {
		normalizedLimit = maxListLimit
	}
	return normalizedCursor, normalizedLimit
}

func cursorAfterKey(cursor string) kv.Key {
	if cursor == "" {
		return nil
	}
	return append(append(kv.Key{}, rootKey...), cursor)
}

func escapeStoreSegment(value string) string {
	return url.QueryEscape(value)
}

func unescapeStoreSegment(value string) string {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}
