package testutil

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
)

const (
	SeedCredentialName      = "ui-seed-credential"
	SeedOpenAITenantName    = "ui-seed-openai-tenant"
	SeedGeminiTenantName    = "ui-seed-gemini-tenant"
	SeedDashScopeTenantName = "ui-seed-dashscope-tenant"
	SeedModelID             = "ui-seed-openai-chat"
	SeedFirmwareName        = "ui-seed-devkit"
	SeedACLViewName         = "under-12"
	SeedMiniMaxTenantName   = "ui-seed-tenant"
	SeedVoiceID             = "ui-seed-voice"
	SeedVolcCredentialName  = "ui-seed-volc-credential"
	SeedVolcTenantName      = "ui-seed-volc-tenant"
	SeedVolcVoiceID         = "volc-tenant:ui-seed-volc-tenant:ICL_ui_seed_voice"
	SeedWorkflowName        = "ui-seed-workflow"
	SeedWorkspaceName       = "ui-seed-workspace"
)

//go:embed gizclaw_seed_data/**/*.json
var seedFS embed.FS

type RegistrationSeed struct {
	Device apitypes.DeviceInfo `json:"device"`
}

func LoadRegistrationSeed(name string) (RegistrationSeed, error) {
	var seed RegistrationSeed
	if err := readSeedJSON("gizclaw_seed_data/registrations/"+name+".json", &seed); err != nil {
		return RegistrationSeed{}, err
	}
	return seed, nil
}

func RegistrationRequest(_ string, seed RegistrationSeed) gearservice.RegistrationRequest {
	return gearservice.RegistrationRequest{
		Device: seed.Device,
	}
}

func LoadDeviceConfigSeed() (apitypes.Configuration, error) {
	var config apitypes.Configuration
	if err := readSeedJSON("gizclaw_seed_data/gear_config/device.json", &config); err != nil {
		return apitypes.Configuration{}, err
	}
	return config, nil
}

func LoadAdminCatalogSeed() (apitypes.Resource, error) {
	var resource apitypes.Resource
	if err := readSeedJSON("gizclaw_seed_data/resources/admin_catalog.json", &resource); err != nil {
		return apitypes.Resource{}, err
	}
	return resource, nil
}

func ApplyAdminCatalogSeed(ctx context.Context, api *adminservice.ClientWithResponses) error {
	resource, err := LoadAdminCatalogSeed()
	if err != nil {
		return err
	}
	resp, err := api.ApplyResourceWithResponse(ctx, resource)
	if err != nil {
		return err
	}
	if resp.JSON200 == nil {
		return seedResponseError("apply admin catalog", resp.StatusCode(), resp.Body, resp.JSON400, resp.JSON409, resp.JSON500, resp.JSON501)
	}
	return nil
}

func ApplyWorkspaceSeed(ctx context.Context, api *adminservice.ClientWithResponses) error {
	var workflowResource apitypes.Resource
	if err := readSeedJSON("gizclaw_seed_data/resources/workflow.json", &workflowResource); err != nil {
		return err
	}
	workflow, err := workflowResource.AsWorkflowResource()
	if err != nil {
		return err
	}
	workflowResp, err := api.PutWorkflowWithResponse(ctx, workflow.Metadata.Name, workflow.Spec)
	if err != nil {
		return err
	}
	if workflowResp.JSON200 == nil {
		return seedResponseError("put workflow", workflowResp.StatusCode(), workflowResp.Body, workflowResp.JSON400, workflowResp.JSON500)
	}

	var workspaceResource apitypes.Resource
	if err := readSeedJSON("gizclaw_seed_data/resources/workspace.json", &workspaceResource); err != nil {
		return err
	}
	workspace, err := workspaceResource.AsWorkspaceResource()
	if err != nil {
		return err
	}
	workspaceResp, err := api.PutWorkspaceWithResponse(ctx, workspace.Metadata.Name, adminservice.WorkspaceUpsert{
		Name:         workspace.Metadata.Name,
		Parameters:   workspace.Spec.Parameters,
		WorkflowName: workspace.Spec.WorkflowName,
	})
	if err != nil {
		return err
	}
	if workspaceResp.JSON200 == nil {
		return seedResponseError("put workspace", workspaceResp.StatusCode(), workspaceResp.Body, workspaceResp.JSON400, workspaceResp.JSON500)
	}
	return nil
}

func ApplyDeviceConfigSeed(ctx context.Context, api *adminservice.ClientWithResponses, publicKey string) error {
	config, err := LoadDeviceConfigSeed()
	if err != nil {
		return err
	}
	resp, err := api.PutPeerConfigWithResponse(ctx, publicKey, config)
	if err != nil {
		return err
	}
	if resp.JSON200 == nil {
		return seedResponseError("put peer config", resp.StatusCode(), resp.Body, resp.JSON400, resp.JSON404)
	}
	return nil
}

func CopyAdminCatalogSeedJSON(w io.Writer) error {
	data, err := seedFS.ReadFile("gizclaw_seed_data/resources/admin_catalog.json")
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readSeedJSON(path string, target interface{}) error {
	data, err := seedFS.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func seedResponseError(action string, status int, body []byte, payloads ...interface{}) error {
	for _, payload := range payloads {
		if errorPayload, ok := payload.(*apitypes.ErrorResponse); ok && errorPayload != nil {
			return fmt.Errorf("%s failed: %s: %s", action, errorPayload.Error.Code, errorPayload.Error.Message)
		}
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		text = http.StatusText(status)
	}
	if text == "" {
		text = "empty response"
	}
	return fmt.Errorf("%s failed with status %d: %s", action, status, text)
}
