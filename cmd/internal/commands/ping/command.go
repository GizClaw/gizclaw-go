package pingcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var ctxName string

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping the server (peer-layer time sync)",
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
