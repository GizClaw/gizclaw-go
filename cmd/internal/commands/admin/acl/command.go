package aclcmd

import (
	"context"
	"encoding/json"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var ctxName string
	cmd := &cobra.Command{
		Use:   "acl",
		Short: "Read ACL resources",
	}
	cmd.PersistentFlags().StringVar(&ctxName, "context", "", "context name (default: current)")
	cmd.AddCommand(newViewsCmd(&ctxName))
	return cmd
}

func newViewsCmd(ctxName *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "views",
		Short: "Read ACL views",
	}
	cmd.AddCommand(
		newListViewsCmd(ctxName),
		newGetViewCmd(ctxName),
	)
	return cmd
}

func newListViewsCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List ACL views",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			items, err := client.ListACLViews(context.Background(), c)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
		},
	}
}

func newGetViewCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get an ACL view",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.GetACLView(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
}
