package server

import (
	"errors"
	"fmt"

	"github.com/GizClaw/gizclaw-go/cmd/internal/storage"
	"github.com/GizClaw/gizclaw-go/cmd/internal/stores"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw"
)

var BuildCommit = "dev"

// CmdServer owns the command-layer store registry for a gizclaw server.
type CmdServer struct {
	*gizclaw.Server
	stores *stores.Stores
}

func (s *CmdServer) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.Server != nil {
		errs = append(errs, s.Server.Close())
		s.Server = nil
	}
	if s.stores != nil {
		errs = append(errs, s.stores.Close())
		s.stores = nil
	}
	return errors.Join(errs...)
}

// New wires an already prepared in-memory config into a command server.
func New(cfg Config) (srv *CmdServer, err error) {
	cfg, err = prepareConfig(cfg)
	if err != nil {
		return nil, err
	}
	ss, err := newStoreRegistry(cfg)
	if err != nil {
		return nil, fmt.Errorf("server: stores: %w", err)
	}
	openedStores := ss
	defer func() {
		if err != nil {
			err = errors.Join(err, openedStores.Close())
		}
	}()

	gearsKV, err := ss.KV(cfg.Gears.Store)
	if err != nil {
		return nil, fmt.Errorf("server: gears store: %w", err)
	}

	fwStore, err := ss.DepotStore(cfg.Depots.Store)
	if err != nil {
		return nil, fmt.Errorf("server: firmware store: %w", err)
	}

	gizServer := &gizclaw.Server{
		KeyPair:         cfg.KeyPair,
		ListenAddr:      cfg.ListenAddr,
		GearStore:       gearsKV,
		BuildCommit:     BuildCommit,
		ServerPublicKey: cfg.KeyPair.Public,
		AdminPublicKey:  cfg.AdminPublicKey,
		DepotStore:      fwStore,
	}
	if len(cfg.Storage) > 0 {
		if gizServer.CredentialStore, err = ss.KV(cfg.Credentials.Store); err != nil {
			return nil, fmt.Errorf("server: credentials store: %w", err)
		}
		if gizServer.MiniMaxCredentialStore, err = ss.KV(cfg.MiniMax.CredentialsStore); err != nil {
			return nil, fmt.Errorf("server: minimax credentials store: %w", err)
		}
		if gizServer.MiniMaxTenantStore, err = ss.KV(cfg.MiniMax.TenantsStore); err != nil {
			return nil, fmt.Errorf("server: minimax tenants store: %w", err)
		}
		if gizServer.VoiceStore, err = ss.KV(cfg.MiniMax.VoicesStore); err != nil {
			return nil, fmt.Errorf("server: voices store: %w", err)
		}
		if gizServer.WorkspaceStore, err = ss.KV(cfg.Workspaces.Store); err != nil {
			return nil, fmt.Errorf("server: workspaces store: %w", err)
		}
		if gizServer.WorkspaceTemplateStore, err = ss.KV(cfg.Workspaces.TemplatesStore); err != nil {
			return nil, fmt.Errorf("server: workspace template reference store: %w", err)
		}
		if gizServer.DepotMetadataStore, err = ss.KV(cfg.Depots.MetadataStore); err != nil {
			return nil, fmt.Errorf("server: firmware metadata store: %w", err)
		}
		if gizServer.TemplateStore, err = ss.KV(cfg.WorkspaceTemplates.Store); err != nil {
			return nil, fmt.Errorf("server: workspace templates store: %w", err)
		}
	}
	return &CmdServer{Server: gizServer, stores: ss}, nil
}

func newStoreRegistry(cfg Config) (*stores.Stores, error) {
	if len(cfg.Storage) == 0 {
		return stores.New(cfg.Stores)
	}
	physical, err := storage.New(cfg.Storage)
	if err != nil {
		return nil, err
	}
	ss, err := stores.NewWithOwnedStorage(physical, cfg.Stores)
	if err != nil {
		return nil, err
	}
	return ss, nil
}
