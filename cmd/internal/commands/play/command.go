package playcmd

import (
	"strings"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/spf13/cobra"
)

var listenAndServePlayUI = client.ListenAndServePlayUI

func NewCmd() *cobra.Command {
	var ctxName string
	var listenAddr string
	cmd := &cobra.Command{
		Use:   "play",
		Short: "Open the Play web UI",
		Long:  "Open the Play web UI. When --listen is set, the current context is automatically registered before the UI starts.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(listenAddr) == "" {
				return cmd.Help()
			}
			return listenAndServePlayUI(ctxName, listenAddr, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	cmd.Flags().StringVar(&listenAddr, "listen", "", "listen address or port for the play web UI")
	return cmd
}
