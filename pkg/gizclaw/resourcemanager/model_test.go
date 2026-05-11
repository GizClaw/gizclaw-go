package resourcemanager

import (
	"context"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/modelcatalog"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

func TestApplyModelCreatesUpdatesAndSkipsUnchanged(t *testing.T) {
	manager := newModelManager()
	resource := mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Model",
		"metadata": {"name": "qwen-flash"},
		"spec": {
			"kind": "llm",
			"provider": {"kind": "openai-compatible", "name": "dashscope"},
			"source": "manual",
			"name": "Qwen Flash"
		}
	}`)

	result, err := manager.Apply(context.Background(), resource)
	if err != nil {
		t.Fatalf("Apply(create Model) error = %v", err)
	}
	if result.Action != apitypes.ApplyActionCreated {
		t.Fatalf("Apply(create Model) action = %s", result.Action)
	}
	result, err = manager.Apply(context.Background(), resource)
	if err != nil {
		t.Fatalf("Apply(unchanged Model) error = %v", err)
	}
	if result.Action != apitypes.ApplyActionUnchanged {
		t.Fatalf("Apply(unchanged Model) action = %s", result.Action)
	}

	updated := mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Model",
		"metadata": {"name": "qwen-flash"},
		"spec": {
			"kind": "llm",
			"provider": {"kind": "openai-compatible", "name": "dashscope"},
			"source": "manual",
			"name": "Qwen Flash",
			"description": "fast model"
		}
	}`)
	result, err = manager.Apply(context.Background(), updated)
	if err != nil {
		t.Fatalf("Apply(update Model) error = %v", err)
	}
	if result.Action != apitypes.ApplyActionUpdated {
		t.Fatalf("Apply(update Model) action = %s", result.Action)
	}
}

func TestPutGetDeleteModelResource(t *testing.T) {
	manager := newModelManager()
	resource := mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Model",
		"metadata": {"name": "speech"},
		"spec": {
			"kind": "tts",
			"provider": {"kind": "openai-compatible", "name": "openai"},
			"source": "manual",
			"provider_data": {"openai-compatible": {"upstream_model": "gpt-4o-mini-tts"}}
		}
	}`)

	stored, err := manager.Put(context.Background(), resource)
	if err != nil {
		t.Fatalf("Put(Model) error = %v", err)
	}
	model, err := stored.AsModelResource()
	if err != nil {
		t.Fatalf("AsModelResource(Put) error = %v", err)
	}
	if model.Spec.Kind != apitypes.ModelKindTts {
		t.Fatalf("Put(Model) kind = %s", model.Spec.Kind)
	}

	got, err := manager.Get(context.Background(), apitypes.ResourceKindModel, "speech")
	if err != nil {
		t.Fatalf("Get(Model) error = %v", err)
	}
	gotModel, err := got.AsModelResource()
	if err != nil {
		t.Fatalf("AsModelResource(Get) error = %v", err)
	}
	if gotModel.Metadata.Name != "speech" {
		t.Fatalf("Get(Model) metadata.name = %s", gotModel.Metadata.Name)
	}

	deleted, err := manager.Delete(context.Background(), apitypes.ResourceKindModel, "speech")
	if err != nil {
		t.Fatalf("Delete(Model) error = %v", err)
	}
	deletedModel, err := deleted.AsModelResource()
	if err != nil {
		t.Fatalf("AsModelResource(Delete) error = %v", err)
	}
	if deletedModel.Metadata.Name != "speech" {
		t.Fatalf("Delete(Model) metadata.name = %s", deletedModel.Metadata.Name)
	}
	_, err = manager.Get(context.Background(), apitypes.ResourceKindModel, "speech")
	assertResourceError(t, err, 404, "RESOURCE_NOT_FOUND")
}

func TestModelServiceResponseErrors(t *testing.T) {
	manager := New(Services{Models: errorModelService{}})
	_, _, err := manager.getModel(context.Background(), "model")
	assertResourceError(t, err, 500, "INTERNAL_ERROR")
	for _, tc := range []struct {
		status int
		code   string
	}{
		{status: 400, code: "INVALID_MODEL"},
		{status: 409, code: "MODEL_CONFLICT"},
		{status: 500, code: "INTERNAL_ERROR"},
	} {
		t.Run("put", func(t *testing.T) {
			manager := New(Services{Models: errorModelService{putStatus: tc.status}})
			err := manager.putModel(context.Background(), "model", adminservice.ModelUpsert{})
			assertResourceError(t, err, tc.status, tc.code)
		})
	}
	_, _, err = manager.deleteModel(context.Background(), "model")
	assertResourceError(t, err, 500, "INTERNAL_ERROR")
}

func newModelManager() *Manager {
	return New(Services{
		Models: &modelcatalog.Server{Store: kv.NewMemory(nil)},
	})
}

type errorModelService struct {
	putStatus int
}

func (e errorModelService) CreateModel(context.Context, adminservice.CreateModelRequestObject) (adminservice.CreateModelResponseObject, error) {
	return adminservice.CreateModel500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", "failed")), nil
}

func (e errorModelService) ListModels(context.Context, adminservice.ListModelsRequestObject) (adminservice.ListModelsResponseObject, error) {
	return adminservice.ListModels500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", "failed")), nil
}

func (e errorModelService) DeleteModel(context.Context, adminservice.DeleteModelRequestObject) (adminservice.DeleteModelResponseObject, error) {
	return adminservice.DeleteModel500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", "failed")), nil
}

func (e errorModelService) GetModel(context.Context, adminservice.GetModelRequestObject) (adminservice.GetModelResponseObject, error) {
	return adminservice.GetModel500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", "failed")), nil
}

func (e errorModelService) PutModel(context.Context, adminservice.PutModelRequestObject) (adminservice.PutModelResponseObject, error) {
	switch e.putStatus {
	case 400:
		return adminservice.PutModel400JSONResponse(apitypes.NewErrorResponse("INVALID_MODEL", "invalid")), nil
	case 409:
		return adminservice.PutModel409JSONResponse(apitypes.NewErrorResponse("MODEL_CONFLICT", "conflict")), nil
	default:
		return adminservice.PutModel500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", "failed")), nil
	}
}
