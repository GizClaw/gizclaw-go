package agenthost

import (
	"context"
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkg/genx"
)

var _ genx.Transformer = (*Host)(nil)

type Host struct {
	Resolver    Resolver
	Registry    *Registry
	Coordinator Coordinator
}

func New(resolver Resolver) *Host {
	return &Host{
		Resolver:    resolver,
		Registry:    NewRegistry(),
		Coordinator: NewMemoryCoordinator(),
	}
}

func (h *Host) Register(agentType string, factory Factory) error {
	registry := h.registry()
	if registry == nil {
		return fmt.Errorf("agenthost: registry is required")
	}
	return registry.Register(agentType, factory)
}

func (h *Host) Transform(ctx context.Context, pattern string, input genx.Stream) (genx.Stream, error) {
	if h == nil {
		return nil, fmt.Errorf("agenthost: host is nil")
	}
	if input == nil {
		return nil, fmt.Errorf("agenthost: input stream is required")
	}
	if h.Resolver == nil {
		return nil, fmt.Errorf("agenthost: resolver is required")
	}

	spec, err := h.Resolver.Resolve(ctx, pattern)
	if err != nil {
		return nil, err
	}
	coordinator := h.coordinator()
	if coordinator == nil {
		return nil, fmt.Errorf("agenthost: coordinator is required")
	}
	workspaceName := string(spec.Workspace.Name)
	if workspaceName == "" {
		return nil, fmt.Errorf("agenthost: resolved workspace name is required")
	}
	lease, err := coordinator.Acquire(ctx, workspaceName)
	if err != nil {
		return nil, err
	}

	release := func() {
		_ = lease.Release(context.Background())
	}
	factory, ok := h.registry().Get(spec.AgentType)
	if !ok {
		release()
		return nil, fmt.Errorf("agenthost: agent factory not found for %q", spec.AgentType)
	}
	agent, err := factory.NewAgent(ctx, spec)
	if err != nil {
		release()
		return nil, err
	}
	if agent == nil {
		release()
		return nil, fmt.Errorf("agenthost: factory %q returned nil agent", spec.AgentType)
	}
	output, err := agent.Transform(ctx, pattern, input)
	if err != nil {
		release()
		return nil, err
	}
	if output == nil {
		release()
		return nil, fmt.Errorf("agenthost: agent %q returned nil stream", spec.AgentType)
	}
	return &leaseStream{Stream: output, release: release}, nil
}

func (h *Host) registry() *Registry {
	if h == nil {
		return nil
	}
	if h.Registry == nil {
		h.Registry = NewRegistry()
	}
	return h.Registry
}

func (h *Host) coordinator() Coordinator {
	if h == nil {
		return nil
	}
	if h.Coordinator == nil {
		h.Coordinator = NewMemoryCoordinator()
	}
	return h.Coordinator
}
