package gizclaw

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/resourcemanager"
)

func TestAdminServiceApplyResourceRequiresBody(t *testing.T) {
	t.Parallel()

	resp, err := (&adminService{}).ApplyResource(context.Background(), adminservice.ApplyResourceRequestObject{})
	if err != nil {
		t.Fatalf("ApplyResource() error = %v", err)
	}
	got, ok := resp.(adminservice.ApplyResource400JSONResponse)
	if !ok {
		t.Fatalf("ApplyResource() response = %T", resp)
	}
	if got.Error.Code != "INVALID_RESOURCE" {
		t.Fatalf("ApplyResource() code = %q", got.Error.Code)
	}
}

func TestAdminServiceResourceMethodsHandleValidationAndManagerErrors(t *testing.T) {
	resource := mustPeerServiceResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Credential",
		"metadata": {"name": "minimax-main"},
		"spec": {
			"provider": "minimax",
			"method": "api_key",
			"body": {"api_key": "secret"}
		}
	}`)
	service := &adminService{}

	applyResp, err := service.ApplyResource(context.Background(), adminservice.ApplyResourceRequestObject{JSONBody: &resource})
	if err != nil {
		t.Fatalf("ApplyResource() error = %v", err)
	}
	if got, ok := applyResp.(adminservice.ApplyResource500JSONResponse); !ok || got.Error.Code != "RESOURCE_MANAGER_NOT_CONFIGURED" {
		t.Fatalf("ApplyResource() response = %T %+v", applyResp, applyResp)
	}

	getResp, err := service.GetResource(context.Background(), adminservice.GetResourceRequestObject{
		Kind: apitypes.ResourceKindCredential,
		Name: "minimax-main",
	})
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if got, ok := getResp.(adminservice.GetResource500JSONResponse); !ok || got.Error.Code != "RESOURCE_MANAGER_NOT_CONFIGURED" {
		t.Fatalf("GetResource() response = %T %+v", getResp, getResp)
	}

	putResp, err := service.PutResource(context.Background(), adminservice.PutResourceRequestObject{})
	if err != nil {
		t.Fatalf("PutResource(nil body) error = %v", err)
	}
	if got, ok := putResp.(adminservice.PutResource400JSONResponse); !ok || got.Error.Code != "INVALID_RESOURCE" {
		t.Fatalf("PutResource(nil body) response = %T %+v", putResp, putResp)
	}

	putResp, err = service.PutResource(context.Background(), adminservice.PutResourceRequestObject{
		Kind:     apitypes.ResourceKindWorkspace,
		Name:     "minimax-main",
		JSONBody: &resource,
	})
	if err != nil {
		t.Fatalf("PutResource(path mismatch) error = %v", err)
	}
	if got, ok := putResp.(adminservice.PutResource400JSONResponse); !ok || got.Error.Code != "INVALID_RESOURCE_PATH" {
		t.Fatalf("PutResource(path mismatch) response = %T %+v", putResp, putResp)
	}

	putResp, err = service.PutResource(context.Background(), adminservice.PutResourceRequestObject{
		Kind:     apitypes.ResourceKindCredential,
		Name:     "minimax-main",
		JSONBody: &resource,
	})
	if err != nil {
		t.Fatalf("PutResource(manager error) error = %v", err)
	}
	if got, ok := putResp.(adminservice.PutResource500JSONResponse); !ok || got.Error.Code != "RESOURCE_MANAGER_NOT_CONFIGURED" {
		t.Fatalf("PutResource(manager error) response = %T %+v", putResp, putResp)
	}

	deleteResp, err := service.DeleteResource(context.Background(), adminservice.DeleteResourceRequestObject{
		Kind: apitypes.ResourceKindCredential,
		Name: "minimax-main",
	})
	if err != nil {
		t.Fatalf("DeleteResource() error = %v", err)
	}
	if got, ok := deleteResp.(adminservice.DeleteResource500JSONResponse); !ok || got.Error.Code != "RESOURCE_MANAGER_NOT_CONFIGURED" {
		t.Fatalf("DeleteResource() response = %T %+v", deleteResp, deleteResp)
	}
}

func TestAdminResourceHelpers(t *testing.T) {
	resource := mustPeerServiceResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Credential",
		"metadata": {"name": "minimax-main"},
		"spec": {
			"provider": "minimax",
			"method": "api_key",
			"body": {"api_key": "secret"}
		}
	}`)

	if err := validateResourcePathMatch(resource, apitypes.ResourceKindCredential, "minimax-main"); err != nil {
		t.Fatalf("validateResourcePathMatch() error = %v", err)
	}
	if err := validateResourcePathMatch(resource, apitypes.ResourceKindWorkspace, "minimax-main"); err == nil || !strings.Contains(err.Error(), "kind") {
		t.Fatalf("validateResourcePathMatch(kind mismatch) error = %v", err)
	}
	if err := validateResourcePathMatch(resource, apitypes.ResourceKindCredential, "other"); err == nil || !strings.Contains(err.Error(), "metadata.name") {
		t.Fatalf("validateResourcePathMatch(name mismatch) error = %v", err)
	}

	status, body := resourceManagerError(&resourcemanager.Error{StatusCode: http.StatusNotFound, Code: "RESOURCE_NOT_FOUND", Message: "missing"})
	if status != http.StatusNotFound || body.Error.Code != "RESOURCE_NOT_FOUND" {
		t.Fatalf("resourceManagerError(resource error) = %d %+v", status, body)
	}
	status, body = resourceManagerError(errors.New("boom"))
	if status != http.StatusInternalServerError || body.Error.Code != "RESOURCE_MANAGER_ERROR" {
		t.Fatalf("resourceManagerError(generic error) = %d %+v", status, body)
	}
}

func TestResource200JSONResponseSerializesResourceUnion(t *testing.T) {
	resource := mustPeerServiceResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Credential",
		"metadata": {"name": "minimax-main"},
		"spec": {
			"provider": "minimax",
			"method": "api_key",
			"body": {"api_key": "secret"}
		}
	}`)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/resource", func(ctx *fiber.Ctx) error {
		return resource200JSONResponse{Resource: resource}.VisitGetResourceResponse(ctx)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()
	fiberHTTPHandler(app).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"kind":"Credential"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func mustPeerServiceResource(t *testing.T, raw string) apitypes.Resource {
	t.Helper()

	var resource apitypes.Resource
	if err := json.Unmarshal([]byte(raw), &resource); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return resource
}
