package commands

import (
	"os"
	"strings"

	admincmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/admin"
	contextcmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/context"
	genkeycmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/genkey"
	migratecmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/migrate"
	peercmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/peer"
	playcmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/play"
	servecmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/serve"
	servicecmd "github.com/GizClaw/gizclaw-go/cmd/internal/commands/service"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "gizclaw",
		Short: "GizClaw - peer-to-peer toy network",
	}
	root.SetArgs(normalizeLegacyLongFlags(os.Args[1:]))

	root.AddCommand(
		servecmd.NewCmd(),
		servicecmd.NewCmd(),
		contextcmd.NewCmd(),
		genkeycmd.NewCmd(),
		migratecmd.NewCmd(),
		peercmd.NewCmd(),
		admincmd.NewCmd(),
		playcmd.NewCmd(),
	)

	return root
}

func normalizeLegacyLongFlags(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		if len(arg) > 2 && strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			normalized = append(normalized, "-"+arg)
			continue
		}
		normalized = append(normalized, arg)
	}
	return normalized
}
