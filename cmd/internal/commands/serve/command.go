package servecmd

import (
	"github.com/GizClaw/gizclaw-go/cmd/internal/server"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "serve <dir>",
		Short: "Serve a workspace",
		Long:  "Direct foreground server starts are disabled by default. Install and start the fixed workspace through 'gizclaw service', or pass --force for explicit local/e2e foreground serve.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.ServeWithOptions(args[0], server.ServeOptions{
				Force: force,
			})
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "explicitly allow foreground local serve and replace stale pid files")
	return cmd
}
