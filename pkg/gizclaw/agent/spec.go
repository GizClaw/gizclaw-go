package agent

import (
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

func resolveWorkflowType(workflow apitypes.WorkflowDocument) (string, error) {
	workflowType := strings.TrimSpace(string(workflow.Spec.Driver))
	if workflowType == "" {
		return "", errors.New("agent: workflow spec.driver is required")
	}
	if !workflow.Spec.Driver.Valid() {
		return "", fmt.Errorf("agent: unsupported workflow spec.driver %q", workflow.Spec.Driver)
	}
	return workflowType, nil
}
