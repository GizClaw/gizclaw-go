package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

// Spec is the fully resolved workspace and workflow configuration used to
// construct one per-connection agent runtime.
type Spec struct {
	Workspace    apitypes.Workspace
	Workflow     apitypes.WorkflowDocument
	WorkflowType string
}

type workflowEnvelope struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

func resolveWorkflowType(workflow apitypes.WorkflowDocument) (string, error) {
	data, err := json.Marshal(workflow)
	if err != nil {
		return "", fmt.Errorf("agent: encode workflow: %w", err)
	}
	var env workflowEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return "", fmt.Errorf("agent: decode workflow envelope: %w", err)
	}
	apiVersion := strings.TrimSpace(env.APIVersion)
	if apiVersion == "" {
		return "", errors.New("agent: workflow apiVersion is required")
	}
	group, _, ok := strings.Cut(apiVersion, "/")
	if !ok || strings.TrimSpace(group) == "" {
		return "", fmt.Errorf("agent: unsupported workflow apiVersion %q", apiVersion)
	}
	group = strings.TrimSpace(group)
	if strings.HasPrefix(group, "gizclaw.") {
		group = strings.TrimPrefix(group, "gizclaw.")
	}
	if group == "" {
		return "", fmt.Errorf("agent: unsupported workflow apiVersion %q", apiVersion)
	}
	return group, nil
}
