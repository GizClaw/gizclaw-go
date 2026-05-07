package admincmd

import (
	"strings"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	credentialscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/credentials"
	firmwarecmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/firmware"
	gearscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/gears"
	minimaxtenantscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/minimaxtenants"
	voicescmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/voices"
	volctenantscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/volctenants"
	workspacescmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/workspaces"
	workspacetemplatescmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/workspacetemplates"
	"github.com/spf13/cobra"
)

var listenAndServeAdminUI = client.ListenAndServeAdminUI

func NewCmd() *cobra.Command {
	var ctxName string
	var listenAddr string
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin control-plane commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(listenAddr) == "" {
				return cmd.Help()
			}
			return listenAndServeAdminUI(ctxName, listenAddr, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	cmd.Flags().StringVar(&listenAddr, "listen", "", "listen address or port for the admin web UI")
	cmd.AddCommand(
		newApplyCmd(&ctxName),
		newDeleteCmd(&ctxName),
		newShowCmd(&ctxName),
		gearscmd.NewCmd(),
		firmwarecmd.NewCmd(),
		credentialscmd.NewCmd(),
		minimaxtenantscmd.NewCmd(),
		volctenantscmd.NewCmd(),
		voicescmd.NewCmd(),
		workspacetemplatescmd.NewCmd(),
		workspacescmd.NewCmd(),
	)
	return cmd
}
