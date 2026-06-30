package genkey

import (
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gen-key",
		Short: "Generate a GizClaw private key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			keyPair, err := giznet.GenerateKeyPair()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), keyPair.Private.String())
			return err
		},
	}
}
