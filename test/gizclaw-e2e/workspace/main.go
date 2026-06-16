package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpcapi"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gizcli"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "workspacetest: %v\n", err)
		os.Exit(1)
	}
}

var (
	dialClientForRun      = dialClient
	ensureWorkspaceForRun = func(ctx context.Context, client *gizcli.Client, cfg config) error {
		return ensureWorkspace(ctx, client, cfg)
	}
	selectAndReloadAgentForRun = func(ctx context.Context, client *gizcli.Client, cfg config) error {
		return selectAndReloadAgent(ctx, client, cfg)
	}
	newChatTransportForRun = newChatTransport
	runPersonaDriverForRun = (*personaDriver).run
)

func run(args []string) error {
	var configPath string
	var contextConfigPath string
	flags := flag.NewFlagSet("workspacetest", flag.ContinueOnError)
	flags.StringVar(&configPath, "config", "", "workspacetest config path")
	flags.StringVar(&contextConfigPath, "context-config", "", "setup-generated context config path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(configPath, contextConfigPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	client, serveDone, err := dialClientForRun(cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Close()
		<-serveDone
	}()

	if err := ensureWorkspaceForRun(ctx, client, cfg); err != nil {
		return err
	}

	openaiClient := openai.NewClient(
		option.WithAPIKey("gizclaw-peer"),
		option.WithBaseURL("http://gizclaw/v1"),
		option.WithHTTPClient(client.HTTPClient(gizcli.ServiceOpenAI)),
	)
	driver := &personaDriver{
		cfg:    cfg,
		client: openaiClient,
		newTransport: func() (*chatTransport, error) {
			return newChatTransportForRun(client)
		},
		reloadAgent: func(ctx context.Context) error {
			return selectAndReloadAgentForRun(ctx, client, cfg)
		},
	}
	defer driver.close()
	stats, err := runPersonaDriverForRun(driver, ctx)
	printRunSummary(cfg, stats)
	return err
}

func dialClient(cfg config) (*gizcli.Client, <-chan error, error) {
	keyPair, err := parsePrivateKey(cfg.ClientPrivateKey)
	if err != nil {
		return nil, nil, err
	}
	serverPK, err := parsePublicKey(cfg.Server.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	client := &gizcli.Client{
		KeyPair:    keyPair,
		CipherMode: giznet.CipherMode(cfg.Server.CipherMode),
	}
	if err := client.Dial(serverPK, cfg.Server.Addr); err != nil {
		return nil, nil, err
	}
	done := make(chan error, 1)
	go func() {
		done <- client.Serve()
	}()
	return client, done, nil
}

type runControlClient interface {
	CreateWorkflow(context.Context, string, rpcapi.WorkflowCreateRequest) (*rpcapi.WorkflowCreateResponse, error)
	PutWorkflow(context.Context, string, rpcapi.WorkflowPutRequest) (*rpcapi.WorkflowPutResponse, error)
	CreateWorkspace(context.Context, string, rpcapi.WorkspaceCreateRequest) (*rpcapi.WorkspaceCreateResponse, error)
	PutWorkspace(context.Context, string, rpcapi.WorkspacePutRequest) (*rpcapi.WorkspacePutResponse, error)
	SetServerRunAgent(context.Context, string, rpcapi.ServerSetRunAgentRequest) (*rpcapi.ServerSetRunAgentResponse, error)
	ReloadServerRun(context.Context, string) (*rpcapi.ServerReloadRunResponse, error)
	GetServerRunStatus(context.Context, string, ...rpcapi.ServerGetRunStatusRequest) (*rpcapi.ServerGetRunStatusResponse, error)
}

func ensureWorkspace(ctx context.Context, client runControlClient, cfg config) error {
	workflow := workflowDocument(cfg)
	if _, err := client.CreateWorkflow(ctx, "workspacetest.workflow.create", workflow); err != nil {
		if !isRPCConflict(err) {
			return fmt.Errorf("create workflow %q: %w", cfg.Workflow.Name, err)
		}
		if _, err := client.PutWorkflow(ctx, "workspacetest.workflow.put", rpcapi.WorkflowPutRequest{
			Name: cfg.Workflow.Name,
			Body: workflow,
		}); err != nil {
			return fmt.Errorf("update workflow %q: %w", cfg.Workflow.Name, err)
		}
	}

	workspace := workspaceDocument(cfg)
	if _, err := client.CreateWorkspace(ctx, "workspacetest.workspace.create", workspace); err != nil {
		if !isRPCConflict(err) {
			return fmt.Errorf("create workspace %q: %w", cfg.Workspace, err)
		}
		if _, err := client.PutWorkspace(ctx, "workspacetest.workspace.put", rpcapi.WorkspacePutRequest{
			Name: cfg.Workspace,
			Body: workspace,
		}); err != nil {
			return fmt.Errorf("update workspace %q: %w", cfg.Workspace, err)
		}
	}
	return nil
}

func workflowDocument(cfg config) rpcapi.WorkflowCreateRequest {
	description := cfg.Workflow.Description
	if description == "" {
		description = "Doubao realtime e2e workspace workflow"
	}
	spec := map[string]interface{}{
		"realtime_model": cfg.Workflow.RealtimeModel,
		"realtime": map[string]interface{}{
			"session": map[string]interface{}{
				"auth_mode":     cfg.Workflow.Session.AuthMode,
				"bot_name":      cfg.Workflow.Session.BotName,
				"model":         cfg.Workflow.Session.Model,
				"resource_id":   cfg.Workflow.Session.ResourceID,
				"system_role":   cfg.Workflow.Session.SystemRole,
				"vad_window_ms": cfg.Workflow.Session.VADWindowMS,
			},
			"output": map[string]interface{}{
				"speaker": cfg.Workflow.Output.Speaker,
			},
		},
	}
	return rpcapi.WorkflowCreateRequest{
		ApiVersion: "gizclaw.flowcraft/v1alpha1",
		Kind:       "FlowcraftWorkflow",
		Metadata: rpcapi.WorkflowMetadata{
			Name:        cfg.Workflow.Name,
			Description: &description,
		},
		Spec: spec,
	}
}

func workspaceDocument(cfg config) rpcapi.WorkspaceCreateRequest {
	params := map[string]interface{}{
		"agent_type":     cfg.Agent,
		"realtime_model": cfg.Workflow.RealtimeModel,
	}
	for key, value := range cfg.Workflow.Parameters {
		params[key] = value
	}
	return rpcapi.WorkspaceCreateRequest{
		Name:         cfg.Workspace,
		WorkflowName: cfg.Workflow.Name,
		Parameters:   &params,
	}
}

func isRPCConflict(err error) bool {
	var rpcErr rpcapi.Error
	return errors.As(err, &rpcErr) && rpcErr.Code == rpcapi.RPCErrorCodeConflict
}

func selectAndReloadAgent(ctx context.Context, client runControlClient, cfg config) error {
	selection := rpcapi.ServerSetRunAgentRequest{WorkspaceName: cfg.Workspace}
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := client.SetServerRunAgent(ctx, "workspacetest.run.agent.set", selection); err != nil {
			return fmt.Errorf("select workspace %q: %w", cfg.Workspace, err)
		}
		if _, err := client.ReloadServerRun(ctx, "workspacetest.run.reload"); err != nil {
			if !isAgentAlreadyRunning(err) || time.Now().After(deadline) {
				return fmt.Errorf("reload workspace %q: %w", cfg.Workspace, err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}
		status, err := client.GetServerRunStatus(ctx, "workspacetest.run.status")
		if err != nil {
			return fmt.Errorf("get run status: %w", err)
		}
		if status.State == rpcapi.PeerRunStatusStateRunning {
			if status.WorkspaceName == nil || *status.WorkspaceName != cfg.Workspace {
				return fmt.Errorf("running workspace = %v, want %q", status.WorkspaceName, cfg.Workspace)
			}
			return nil
		}
		if status.State == rpcapi.PeerRunStatusStateError {
			message := ""
			if status.Message != nil {
				message = *status.Message
			}
			return fmt.Errorf("workspace %q failed to start: %s", cfg.Workspace, message)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("workspace %q did not reach running state; last=%s", cfg.Workspace, status.State)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func printRunSummary(cfg config, stats []roundStats) {
	fmt.Printf("server=%s workflow=%s workspace=%s agent=%s rounds=%d output_dir=%s\n", cfg.Server.Addr, cfg.Workflow.Name, cfg.Workspace, cfg.Agent, cfg.Rounds, cfg.OutputDir)
	for _, stat := range stats {
		fmt.Printf("round=%d user_chars=%d transcript_chars=%d assistant_chars=%d input_packets=%d input_bytes=%d downlink_packets=%d downlink_bytes=%d events=%d workspace_uplink_send=%s after_eos_transcript_first_chunk=%s transcript_first_before_eos=%t after_eos_text_first_chunk=%s after_eos_audio_first_chunk=%s after_eos_complete=%s workspace_total=%s\n",
			stat.Index,
			runeCount(stat.UserText),
			runeCount(stat.Transcript),
			runeCount(stat.AssistantText),
			stat.InputOpusPackets,
			stat.InputOpusBytes,
			stat.DownlinkPackets,
			stat.DownlinkBytes,
			stat.EventCount,
			stat.UplinkSend.Round(time.Millisecond),
			stat.FirstTranscriptChunk.Round(time.Millisecond),
			stat.FirstTranscriptBeforeEOS,
			stat.FirstAssistantTextChunk.Round(time.Millisecond),
			stat.FirstAudioChunk.Round(time.Millisecond),
			stat.ResponseTotal.Round(time.Millisecond),
			stat.WorkspaceTotal.Round(time.Millisecond),
		)
		fmt.Printf("round_detail=%s\n", encodeJSONLine(map[string]string{
			"user":       stat.UserText,
			"transcript": stat.Transcript,
			"assistant":  stat.AssistantText,
		}))
	}
	fmt.Printf("timing_summary=%s\n", encodeJSONLine(roundTimingSummary(stats)))
}

type timingSummary struct {
	Count int     `json:"count"`
	MinMS float64 `json:"min_ms"`
	AvgMS float64 `json:"avg_ms"`
	P50MS float64 `json:"p50_ms"`
	P95MS float64 `json:"p95_ms"`
	MaxMS float64 `json:"max_ms"`
}

func roundTimingSummary(stats []roundStats) map[string]timingSummary {
	return map[string]timingSummary{
		"workspace_uplink_send":          summarizeDurations(stats, func(s roundStats) time.Duration { return s.UplinkSend }),
		"after_eos_transcript_first":     summarizeDurations(stats, func(s roundStats) time.Duration { return s.FirstTranscriptChunk }),
		"after_eos_text_first":           summarizeDurations(stats, func(s roundStats) time.Duration { return s.FirstAssistantTextChunk }),
		"after_eos_audio_first":          summarizeDurations(stats, func(s roundStats) time.Duration { return s.FirstAudioChunk }),
		"after_eos_complete":             summarizeDurations(stats, func(s roundStats) time.Duration { return s.ResponseTotal }),
		"workspace_total_including_send": summarizeDurations(stats, func(s roundStats) time.Duration { return s.WorkspaceTotal }),
	}
}

func summarizeDurations(stats []roundStats, pick func(roundStats) time.Duration) timingSummary {
	values := make([]float64, 0, len(stats))
	for _, stat := range stats {
		value := pick(stat)
		if value <= 0 {
			continue
		}
		values = append(values, durationMilliseconds(value))
	}
	if len(values) == 0 {
		return timingSummary{}
	}
	sort.Float64s(values)
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return timingSummary{
		Count: len(values),
		MinMS: values[0],
		AvgMS: sum / float64(len(values)),
		P50MS: percentile(values, 0.50),
		P95MS: percentile(values, 0.95),
		MaxMS: values[len(values)-1],
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	index := int(p*float64(len(sorted)-1) + 0.5)
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func durationMilliseconds(value time.Duration) float64 {
	return float64(value.Microseconds()) / 1000
}
