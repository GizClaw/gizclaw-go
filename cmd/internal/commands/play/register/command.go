package playregistercmd

import (
	"context"
	"encoding/json"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var ctxName string
	var req gearservice.RegistrationRequest

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register the current device",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			result, err := client.Register(context.Background(), c, req)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	var name string
	var sn string
	var manufacturer string
	var model string
	var hardwareRevision string
	var depot string
	var firmwareSemver string
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		req.Device = apitypes.DeviceInfo{
			Name: optionalString(name),
			Sn:   optionalString(sn),
			Hardware: &apitypes.HardwareInfo{
				Manufacturer:     optionalString(manufacturer),
				Model:            optionalString(model),
				HardwareRevision: optionalString(hardwareRevision),
				Depot:            optionalString(depot),
				FirmwareSemver:   optionalString(firmwareSemver),
			},
		}
	}
	cmd.Flags().StringVar(&name, "name", "", "device name")
	cmd.Flags().StringVar(&sn, "sn", "", "serial number")
	cmd.Flags().StringVar(&manufacturer, "manufacturer", "", "manufacturer")
	cmd.Flags().StringVar(&model, "model", "", "model")
	cmd.Flags().StringVar(&hardwareRevision, "hardware-revision", "", "hardware revision")
	cmd.Flags().StringVar(&depot, "depot", "", "depot")
	cmd.Flags().StringVar(&firmwareSemver, "firmware-semver", "", "firmware semver")
	return cmd
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
