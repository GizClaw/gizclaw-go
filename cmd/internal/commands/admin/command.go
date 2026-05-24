package admincmd

import (
	"strings"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	aclcmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/acl"
	credentialscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/credentials"
	dashscopetenantscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/dashscopetenants"
	firmwarescmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/firmwares"
	geminitenantscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/geminitenants"
	minimaxtenantscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/minimaxtenants"
	modelscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/models"
	openaitenantscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/openaitenants"
	peerscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/peers"
	voicescmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/voices"
	volctenantscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/volctenants"
	workflowscmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/workflows"
	workspacescmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin/workspaces"
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
		aclcmd.NewCmd(),
		peerscmd.NewCmd(),
		credentialscmd.NewCmd(),
		firmwarescmd.NewCmd(),
		openaitenantscmd.NewCmd(),
		geminitenantscmd.NewCmd(),
		dashscopetenantscmd.NewCmd(),
		minimaxtenantscmd.NewCmd(),
		volctenantscmd.NewCmd(),
		modelscmd.NewCmd(),
		voicescmd.NewCmd(),
		workflowscmd.NewCmd(),
		workspacescmd.NewCmd(),
	)
	return cmd
}
