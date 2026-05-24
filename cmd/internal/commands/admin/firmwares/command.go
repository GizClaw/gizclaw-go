package firmwarescmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var ctxName string
	cmd := &cobra.Command{
		Use:   "firmwares",
		Short: "Manage firmware release lines",
	}
	cmd.PersistentFlags().StringVar(&ctxName, "context", "", "context name (default: current)")
	cmd.AddCommand(
		newListCmd(&ctxName),
		newCreateCmd(&ctxName),
		newGetCmd(&ctxName),
		newPutCmd(&ctxName),
		newDeleteCmd(&ctxName),
		newReleaseCmd(&ctxName),
		newRollbackCmd(&ctxName),
	)
	return cmd
}

func newListCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List firmwares",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			items, err := client.ListFirmwares(context.Background(), c)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
		},
	}
}

func newCreateCmd(ctxName *string) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "create -f <file>",
		Short: "Create a firmware",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := readFirmwareUpsert(cmd, file)
			if err != nil {
				return err
			}
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.CreateFirmware(context.Background(), c, req)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "firmware JSON file, or '-' for stdin")
	return cmd
}

func newGetCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get a firmware",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.GetFirmware(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
}

func newPutCmd(ctxName *string) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "put <name> -f <file>",
		Short: "Create or update a firmware",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := readFirmwareUpsert(cmd, file)
			if err != nil {
				return err
			}
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.PutFirmware(context.Background(), c, args[0], req)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "firmware JSON file, or '-' for stdin")
	return cmd
}

func newDeleteCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a firmware",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.DeleteFirmware(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
}

func newReleaseCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "release <name>",
		Short: "Promote firmware slots",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.ReleaseFirmware(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
}

func newRollbackCmd(ctxName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <name>",
		Short: "Rollback firmware stable slot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			item, err := client.RollbackFirmware(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		},
	}
}

func readFirmwareUpsert(cmd *cobra.Command, file string) (adminservice.FirmwareUpsert, error) {
	if strings.TrimSpace(file) == "" {
		return adminservice.FirmwareUpsert{}, fmt.Errorf("required flag: --file")
	}
	var r io.Reader
	if file == "-" {
		r = cmd.InOrStdin()
	} else {
		f, err := os.Open(file)
		if err != nil {
			return adminservice.FirmwareUpsert{}, err
		}
		defer f.Close()
		r = f
	}
	var req adminservice.FirmwareUpsert
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return adminservice.FirmwareUpsert{}, err
	}
	return req, nil
}
