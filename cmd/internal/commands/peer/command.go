package peercmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpcapi"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Peer connection and self-service commands",
	}
	cmd.AddCommand(
		newPingCmd(),
		newServerInfoCmd(),
		newSetNameCmd(),
		newTestSpeedCmd(),
	)
	return cmd
}

func newPingCmd() *cobra.Command {
	var ctxName string

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping the server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()

			t1 := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			ping, err := c.Ping(ctx, "ping")
			if err != nil {
				return err
			}
			t4 := time.Now()
			rtt := t4.Sub(t1)
			serverTime := time.UnixMilli(ping.ServerTime)
			clientMid := t1.Add(rtt / 2)
			clockDiff := serverTime.Sub(clientMid)

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Server Time: %s\n", serverTime.Format(time.RFC3339Nano))
			fmt.Fprintf(out, "RTT:         %v\n", rtt.Round(time.Microsecond))
			fmt.Fprintf(out, "Clock Diff:  %v\n", clockDiff.Round(time.Microsecond))
			return nil
		},
	}

	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	return cmd
}

func newServerInfoCmd() *cobra.Command {
	var ctxName string
	cmd := &cobra.Command{
		Use:   "server-info",
		Short: "Show server information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			info, err := client.GetServerInfo(context.Background(), c)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	return cmd
}

func newSetNameCmd() *cobra.Command {
	var ctxName string
	cmd := &cobra.Command{
		Use:   "set-name <name>",
		Short: "Set current device name",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("device name must not be empty")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			info, err := client.SetName(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	return cmd
}

func newTestSpeedCmd() *cobra.Command {
	var ctxName string
	var upContentLength int64 = 10 * 1024 * 1024
	var downContentLength int64 = 10 * 1024 * 1024
	var timeout time.Duration = 30 * time.Second

	cmd := &cobra.Command{
		Use:   "test-speed",
		Short: "Measure concurrent upload and download throughput",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			result, err := c.SpeedTest(ctx, "peer.speed_test.run", rpcapi.SpeedTestRequest{
				UpContentLength:   upContentLength,
				DownContentLength: downContentLength,
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Up Bytes:     %d\n", result.UpBytes)
			fmt.Fprintf(out, "Down Bytes:   %d\n", result.DownBytes)
			fmt.Fprintf(out, "Duration:     %v\n", result.Duration.Round(time.Millisecond))
			fmt.Fprintf(out, "Up Speed:     %.2f Mbps\n", result.UpMbps())
			fmt.Fprintf(out, "Down Speed:   %.2f Mbps\n", result.DownMbps())
			return nil
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	cmd.Flags().Int64Var(&upContentLength, "up-content-length", upContentLength, "upload byte count")
	cmd.Flags().Int64Var(&downContentLength, "down-content-length", downContentLength, "download byte count")
	cmd.Flags().DurationVar(&timeout, "timeout", timeout, "speed test timeout")
	return cmd
}
