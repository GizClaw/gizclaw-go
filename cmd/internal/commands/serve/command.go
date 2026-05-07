package servecmd

import (
	"github.com/GizClaw/gizclaw-go/cmd/internal/server"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "serve <dir>",
		Short: "Reject direct server starts",
		Long:  "Direct server starts are disabled. Install and start the fixed workspace through 'gizclaw service' instead.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.ServeWithOptions(args[0], server.ServeOptions{
				Force: force,
			})
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "legacy flag; direct serve still requires gizclaw service")
	return cmd
}
