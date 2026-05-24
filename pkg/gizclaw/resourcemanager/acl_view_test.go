package resourcemanager

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/acl"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	_ "modernc.org/sqlite"
)

func TestApplyACLViewResource(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	server := &acl.Server{DB: db}
	if err := server.Migration(context.Background()); err != nil {
		t.Fatalf("Migration() error = %v", err)
	}
	manager := New(Services{ACL: server})
	resource := mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "ACLView",
		"metadata": {"name": "under-12"},
		"spec": {
			"description": "Content for children under 12."
		}
	}`)

	result, err := manager.Apply(context.Background(), resource)
	if err != nil {
		t.Fatalf("Apply(create) error = %v", err)
	}
	if result.Kind != apitypes.ResourceKindACLView || result.Action != apitypes.ApplyActionCreated {
		t.Fatalf("Apply(create) = %+v", result)
	}
	result, err = manager.Apply(context.Background(), resource)
	if err != nil {
		t.Fatalf("Apply(unchanged) error = %v", err)
	}
	if result.Action != apitypes.ApplyActionUnchanged {
		t.Fatalf("Apply(unchanged) = %+v", result)
	}
	stored, err := manager.Get(context.Background(), apitypes.ResourceKindACLView, "under-12")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	data := string(mustMarshal(t, stored))
	if !strings.Contains(data, `"kind":"ACLView"`) || !strings.Contains(data, `"description":"Content for children under 12."`) {
		t.Fatalf("Get() resource = %s", data)
	}
	deleted, err := manager.Delete(context.Background(), apitypes.ResourceKindACLView, "under-12")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	data = string(mustMarshal(t, deleted))
	if !strings.Contains(data, `"kind":"ACLView"`) || !strings.Contains(data, `"name":"under-12"`) {
		t.Fatalf("Delete() resource = %s", data)
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	return data
}
