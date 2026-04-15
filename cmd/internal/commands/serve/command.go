package servecmd

import (
	"github.com/giztoy/giztoy-go/cmd/internal/server"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve <dir>",
		Short: "Start the Giztoy server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Serve(args[0])
		},
	}
}
