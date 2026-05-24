package gizclaw

import (
	"context"
	"errors"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/acl"
)

type Migrator struct {
	ACL *acl.Server
}

func (m *Migrator) Migrate(ctx context.Context) error {
	if m == nil {
		return errors.New("gizclaw: nil migrator")
	}
	if m.ACL != nil {
		if err := m.ACL.Migration(ctx); err != nil {
			return err
		}
	}
	return nil
}
