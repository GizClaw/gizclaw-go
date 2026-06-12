package connectcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cmdclient "github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpcapi"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gizcli"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to the GizClaw server",
	}
	cmd.AddCommand(
		newPingCmd(),
		newServerInfoCmd(),
		newSetNameCmd(),
		newSayCmd(),
		newTestSpeedCmd(),
		newPetCmd(),
		newWalletCmd(),
		newRewardCmd(),
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
			c, err := cmdclient.ConnectFromContext(ctxName)
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
			c, err := cmdclient.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			info, err := cmdclient.GetServerInfo(context.Background(), c)
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
			c, err := cmdclient.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			info, err := cmdclient.SetName(context.Background(), c, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	return cmd
}

func newSayCmd() *cobra.Command {
	var ctxName string
	var voiceID string
	var timeout time.Duration = 30 * time.Second

	cmd := &cobra.Command{
		Use:   "say --voice <voice-id> <text>",
		Short: "Ask the server to synthesize speech for this connection",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(strings.Join(args, " ")) == "" {
				return fmt.Errorf("text must not be empty")
			}
			if strings.TrimSpace(voiceID) == "" {
				return fmt.Errorf("voice id is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmdclient.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			resp, err := c.ServerRunSay(ctx, "server.run.say", rpcapi.ServerRunSayRequest{
				Text:    strings.Join(args, " "),
				VoiceId: stringPtr(strings.TrimSpace(voiceID)),
			})
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(resp)
		},
	}
	cmd.Flags().StringVar(&ctxName, "context", "", "context name (default: current)")
	cmd.Flags().StringVar(&voiceID, "voice", "", "voice id")
	cmd.Flags().DurationVar(&timeout, "timeout", timeout, "say timeout")
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
			c, err := cmdclient.ConnectFromContext(ctxName)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			result, err := c.SpeedTest(ctx, "all.speed_test.run", rpcapi.SpeedTestRequest{
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

type connectRPCOptions struct {
	contextName string
	timeout     time.Duration
}

func (o *connectRPCOptions) addFlags(cmd *cobra.Command) {
	o.timeout = 30 * time.Second
	cmd.Flags().StringVar(&o.contextName, "context", "", "context name (default: current)")
	cmd.Flags().DurationVar(&o.timeout, "timeout", o.timeout, "RPC timeout")
}

func runConnectJSON(cmd *cobra.Command, opts connectRPCOptions, run func(context.Context, *gizcli.Client) (any, error)) error {
	c, err := cmdclient.ConnectFromContext(opts.contextName)
	if err != nil {
		return err
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()
	result, err := run(ctx, c)
	if err != nil {
		return err
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func nonEmptyFlag(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	return nil
}

func newPetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pet",
		Short: "Manage pets through server RPC",
	}
	cmd.AddCommand(
		newPetListCmd(),
		newPetGetCmd(),
		newPetAdoptCmd(),
		newPetPutCmd(),
		newPetDeleteCmd(),
		newPetActionCmd("feed", "Feed a pet", func(ctx context.Context, c *gizcli.Client, id string, prompt string) (any, error) {
			return c.FeedPet(ctx, "pet.feed", rpcapi.PetFeedRequest{PetId: id, Prompt: prompt})
		}),
		newPetActionCmd("wash", "Wash a pet", func(ctx context.Context, c *gizcli.Client, id string, prompt string) (any, error) {
			return c.WashPet(ctx, "pet.wash", rpcapi.PetWashRequest{PetId: id, Prompt: prompt})
		}),
		newPetActionCmd("play", "Play with a pet", func(ctx context.Context, c *gizcli.Client, id string, prompt string) (any, error) {
			return c.PlayPet(ctx, "pet.play", rpcapi.PetPlayRequest{PetId: id, Prompt: prompt})
		}),
	)
	return cmd
}

func newPetListCmd() *cobra.Command {
	var opts connectRPCOptions
	var cursor string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.ListPets(ctx, "pet.list", rpcapi.PetListRequest{Cursor: optionalString(cursor), Limit: limit})
			})
		},
	}
	opts.addFlags(cmd)
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of pets to return")
	return cmd
}

func newPetGetCmd() *cobra.Command {
	var opts connectRPCOptions
	cmd := &cobra.Command{
		Use:   "get <pet-id>",
		Short: "Get a pet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.GetPet(ctx, "pet.get", rpcapi.PetGetRequest{Id: args[0]})
			})
		},
	}
	opts.addFlags(cmd)
	return cmd
}

func newPetAdoptCmd() *cobra.Command {
	var opts connectRPCOptions
	var id string
	var name string
	cmd := &cobra.Command{
		Use:   "adopt --name <name>",
		Short: "Adopt a pet",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			return nonEmptyFlag("name", name)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.AdoptPet(ctx, "pet.adopt", rpcapi.PetAdoptRequest{Id: optionalString(id), Name: strings.TrimSpace(name)})
			})
		},
	}
	opts.addFlags(cmd)
	cmd.Flags().StringVar(&id, "id", "", "pet id")
	cmd.Flags().StringVar(&name, "name", "", "pet name")
	return cmd
}

func newPetPutCmd() *cobra.Command {
	var opts connectRPCOptions
	var name string
	cmd := &cobra.Command{
		Use:   "put <pet-id> --name <name>",
		Short: "Rename a pet",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			return nonEmptyFlag("name", name)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.PutPet(ctx, "pet.put", rpcapi.PetPutRequest{Id: args[0], Name: strings.TrimSpace(name)})
			})
		},
	}
	opts.addFlags(cmd)
	cmd.Flags().StringVar(&name, "name", "", "pet name")
	return cmd
}

func newPetDeleteCmd() *cobra.Command {
	var opts connectRPCOptions
	cmd := &cobra.Command{
		Use:   "delete <pet-id>",
		Short: "Delete a pet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.DeletePet(ctx, "pet.delete", rpcapi.PetDeleteRequest{Id: args[0]})
			})
		},
	}
	opts.addFlags(cmd)
	return cmd
}

func newPetActionCmd(name string, short string, run func(context.Context, *gizcli.Client, string, string) (any, error)) *cobra.Command {
	var opts connectRPCOptions
	var prompt string
	cmd := &cobra.Command{
		Use:   name + " <pet-id> --prompt <text>",
		Short: short,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			return nonEmptyFlag("prompt", prompt)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return run(ctx, c, args[0], strings.TrimSpace(prompt))
			})
		},
	}
	opts.addFlags(cmd)
	cmd.Flags().StringVar(&prompt, "prompt", "", "action prompt")
	return cmd
}

func newWalletCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "Inspect the current peer wallet",
	}
	cmd.AddCommand(newWalletGetCmd(), newWalletTransactionsCmd())
	return cmd
}

func newWalletGetCmd() *cobra.Command {
	var opts connectRPCOptions
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the current peer wallet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.GetWallet(ctx, "wallet.get", rpcapi.WalletGetRequest{})
			})
		},
	}
	opts.addFlags(cmd)
	return cmd
}

func newWalletTransactionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "Inspect wallet transactions",
	}
	cmd.AddCommand(newWalletTransactionsListCmd(), newWalletTransactionsGetCmd())
	return cmd
}

func newWalletTransactionsListCmd() *cobra.Command {
	var opts connectRPCOptions
	var cursor string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List wallet transactions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.ListWalletTransactions(ctx, "wallet.transactions.list", rpcapi.WalletTransactionsListRequest{Cursor: optionalString(cursor), Limit: limit})
			})
		},
	}
	opts.addFlags(cmd)
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of transactions to return")
	return cmd
}

func newWalletTransactionsGetCmd() *cobra.Command {
	var opts connectRPCOptions
	cmd := &cobra.Command{
		Use:   "get <transaction-id>",
		Short: "Get a wallet transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.GetWalletTransaction(ctx, "wallet.transactions.get", rpcapi.WalletTransactionsGetRequest{Id: args[0]})
			})
		},
	}
	opts.addFlags(cmd)
	return cmd
}

func newRewardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward",
		Short: "Manage rewards through server RPC",
	}
	cmd.AddCommand(newRewardListCmd(), newRewardGetCmd(), newRewardClaimCmd())
	return cmd
}

func newRewardListCmd() *cobra.Command {
	var opts connectRPCOptions
	var cursor string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List rewards",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.ListRewards(ctx, "reward.list", rpcapi.RewardListRequest{Cursor: optionalString(cursor), Limit: limit})
			})
		},
	}
	opts.addFlags(cmd)
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of rewards to return")
	return cmd
}

func newRewardGetCmd() *cobra.Command {
	var opts connectRPCOptions
	cmd := &cobra.Command{
		Use:   "get <reward-id>",
		Short: "Get a reward",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.GetReward(ctx, "reward.get", rpcapi.RewardGetRequest{Id: args[0]})
			})
		},
	}
	opts.addFlags(cmd)
	return cmd
}

func newRewardClaimCmd() *cobra.Command {
	var opts connectRPCOptions
	var prompt string
	cmd := &cobra.Command{
		Use:   "claim --prompt <text>",
		Short: "Claim a prompt-driven reward",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			return nonEmptyFlag("prompt", prompt)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectJSON(cmd, opts, func(ctx context.Context, c *gizcli.Client) (any, error) {
				return c.ClaimReward(ctx, "reward.claim", rpcapi.RewardClaimRequest{Prompt: strings.TrimSpace(prompt)})
			})
		},
	}
	opts.addFlags(cmd)
	cmd.Flags().StringVar(&prompt, "prompt", "", "reward prompt")
	return cmd
}

func stringPtr(value string) *string {
	return &value
}
