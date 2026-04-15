package commands

import (
	admincmd "github.com/giztoy/giztoy-go/cmd/internal/commands/admin"
	contextcmd "github.com/giztoy/giztoy-go/cmd/internal/commands/context"
	pingcmd "github.com/giztoy/giztoy-go/cmd/internal/commands/ping"
	playcmd "github.com/giztoy/giztoy-go/cmd/internal/commands/play"
	servecmd "github.com/giztoy/giztoy-go/cmd/internal/commands/serve"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "giztoy",
		Short: "Giztoy - peer-to-peer toy network",
	}

	root.AddCommand(
		servecmd.NewCmd(),
		contextcmd.NewCmd(),
		pingcmd.NewCmd(),
		admincmd.NewCmd(),
		playcmd.NewCmd(),
	)

	return root
}
