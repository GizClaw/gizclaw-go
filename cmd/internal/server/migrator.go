package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/GizClaw/gizclaw-go/cmd/internal/stores"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/acl"
)

type CmdMigrator struct {
	*gizclaw.Migrator
	stores *stores.Stores
}

func (m *CmdMigrator) Close() error {
	if m == nil {
		return nil
	}
	if m.stores == nil {
		return nil
	}
	err := m.stores.Close()
	m.stores = nil
	return err
}

func NewMigrator(cfg Config) (migrator *CmdMigrator, err error) {
	if cfg.ACL.Store == "" {
		return nil, errors.New("server: acl.store is required")
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

	aclDB, err := ss.SQL(cfg.ACL.Store)
	if err != nil {
		return nil, fmt.Errorf("server: acl store: %w", err)
	}
	return &CmdMigrator{
		Migrator: &gizclaw.Migrator{
			ACL: &acl.Server{DB: aclDB},
		},
		stores: ss,
	}, nil
}

func MigrateWorkspace(ctx context.Context, workspace string) error {
	cfg, err := prepareWorkspaceMigrationConfig(workspace)
	if err != nil {
		return err
	}
	migrator, err := NewMigrator(cfg)
	if err != nil {
		return err
	}
	defer migrator.Close()
	return migrator.Migrate(ctx)
}
