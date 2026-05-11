package workflowscmd

import (
	"context"
	"encoding/json"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var ctxName string
	cmd := &cobra.Command{
		Use:   "workflows",
		Short: "Manage workflows",
	}
	cmd.PersistentFlags().StringVar(&ctxName, "context", "", "context name (default: current)")
	cmd.AddCommand(
		newListCmd(&ctxName),
		newGetCmd(&ctxName),
	)
	return cmd
}

func newListCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			items, err := client.ListWorkflows(context.Background(), c)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
		},
	}
}

func newGetCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.GetWorkflow(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
}
