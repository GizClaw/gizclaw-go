package migratecmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/GizClaw/gizclaw-go/cmd/internal/server"
	"github.com/spf13/cobra"
)

var migrateWorkspace = server.MigrateWorkspace

func NewCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   "migrate --workspace <dir>",
		Short: "Run workspace database migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace = strings.TrimSpace(workspace)
			if workspace == "" {
				return fmt.Errorf("migrate: --workspace is required")
			}
			if err := migrateWorkspace(context.Background(), workspace); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Migrated workspace %s\n", workspace)
			return err
		},
	}
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace directory")
	return cmd
}
