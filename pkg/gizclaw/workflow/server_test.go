package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
	"github.com/gofiber/fiber/v2"
)

func TestServerWorkflowsCRUD(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()

	createDoc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {
			"name": "demo-assistant",
			"description": "flowcraft workflow"
		},
		"spec": {
			"workspace_layout": {},
			"runtime": {},
			"agents": [],
			"entry_agent": ""
		}
	}`)

	createResp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &createDoc})
	if err != nil {
		t.Fatalf("CreateWorkflow() error = %v", err)
	}
	created, ok := createResp.(createWorkflow200Response)
	if !ok {
		t.Fatalf("CreateWorkflow() response = %#v", createResp)
	}
	if got := discriminatorOf(t, created.doc); got != "FlowcraftWorkflow" {
		t.Fatalf("CreateWorkflow() discriminator = %q", got)
	}

	listResp, err := srv.ListWorkflows(ctx, adminservice.ListWorkflowsRequestObject{})
	if err != nil {
		t.Fatalf("ListWorkflows() error = %v", err)
	}
	listed, ok := listResp.(adminservice.ListWorkflows200JSONResponse)
	if !ok {
		t.Fatalf("ListWorkflows() response = %#v", listResp)
	}
	if len(listed.Items) != 1 || listed.HasNext {
		t.Fatalf("ListWorkflows() = %#v", listed)
	}

	getResp, err := srv.GetWorkflow(ctx, adminservice.GetWorkflowRequestObject{Name: "demo-assistant"})
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	gotDoc, ok := getResp.(getWorkflow200Response)
	if !ok {
		t.Fatalf("GetWorkflow() response = %#v", getResp)
	}
	gotSingle := mustSingle(t, gotDoc.doc)
	if gotSingle.Metadata.Name != "demo-assistant" {
		t.Fatalf("GetWorkflow() name = %q", gotSingle.Metadata.Name)
	}

	updateDoc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {
			"name": "demo-assistant",
			"description": "updated description"
		},
		"spec": {
			"runtime": {
				"executor_ref": "local"
			}
		}
	}`)
	putResp, err := srv.PutWorkflow(ctx, adminservice.PutWorkflowRequestObject{
		Name: "demo-assistant",
		Body: &updateDoc,
	})
	if err != nil {
		t.Fatalf("PutWorkflow() error = %v", err)
	}
	putDoc, ok := putResp.(putWorkflow200Response)
	if !ok {
		t.Fatalf("PutWorkflow() response = %#v", putResp)
	}
	putSingle := mustSingle(t, putDoc.doc)
	if putSingle.Metadata.Description == nil || *putSingle.Metadata.Description != "updated description" {
		t.Fatalf("PutWorkflow() description = %#v", putSingle.Metadata.Description)
	}

	deleteResp, err := srv.DeleteWorkflow(ctx, adminservice.DeleteWorkflowRequestObject{Name: "demo-assistant"})
	if err != nil {
		t.Fatalf("DeleteWorkflow() error = %v", err)
	}
	if _, ok := deleteResp.(deleteWorkflow200Response); !ok {
		t.Fatalf("DeleteWorkflow() response = %#v", deleteResp)
	}

	getAfterDelete, err := srv.GetWorkflow(ctx, adminservice.GetWorkflowRequestObject{Name: "demo-assistant"})
	if err != nil {
		t.Fatalf("GetWorkflow() after delete error = %v", err)
	}
	if _, ok := getAfterDelete.(adminservice.GetWorkflow404JSONResponse); !ok {
		t.Fatalf("GetWorkflow() after delete response = %#v", getAfterDelete)
	}
}

func TestServerRejectsUnknownWorkflowKind(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()
	doc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "UnknownWorkflow",
		"metadata": {
			"name": "bad-workflow"
		},
		"spec": {}
	}`)

	resp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc})
	if err != nil {
		t.Fatalf("CreateWorkflow() error = %v", err)
	}
	if _, ok := resp.(adminservice.CreateWorkflow400JSONResponse); !ok {
		t.Fatalf("CreateWorkflow() response = %#v", resp)
	}
}

func TestServerAcceptsEmptyFlowcraftSpec(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()
	doc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {
			"name": "empty-flowcraft"
		},
		"spec": {}
	}`)

	resp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc})
	if err != nil {
		t.Fatalf("CreateWorkflow() error = %v", err)
	}
	created, ok := resp.(createWorkflow200Response)
	if !ok {
		t.Fatalf("CreateWorkflow() response = %#v", resp)
	}
	flowcraft := mustFlowcraft(t, created.doc)
	if flowcraft.Metadata.Name != "empty-flowcraft" {
		t.Fatalf("CreateWorkflow() name = %q", flowcraft.Metadata.Name)
	}
}

func TestServerPutRejectsPathNameMismatch(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()
	doc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {
			"name": "other-name"
		},
		"spec": {}
	}`)

	resp, err := srv.PutWorkflow(ctx, adminservice.PutWorkflowRequestObject{
		Name: "expected-name",
		Body: &doc,
	})
	if err != nil {
		t.Fatalf("PutWorkflow() error = %v", err)
	}
	if _, ok := resp.(adminservice.PutWorkflow400JSONResponse); !ok {
		t.Fatalf("PutWorkflow() response = %#v", resp)
	}

	nilCreateResp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{})
	if err != nil {
		t.Fatalf("CreateWorkflow(nil body) error = %v", err)
	}
	if _, ok := nilCreateResp.(adminservice.CreateWorkflow400JSONResponse); !ok {
		t.Fatalf("CreateWorkflow(nil body) response = %#v", nilCreateResp)
	}

	nilPutResp, err := srv.PutWorkflow(ctx, adminservice.PutWorkflowRequestObject{Name: "expected-name"})
	if err != nil {
		t.Fatalf("PutWorkflow(nil body) error = %v", err)
	}
	if _, ok := nilPutResp.(adminservice.PutWorkflow400JSONResponse); !ok {
		t.Fatalf("PutWorkflow(nil body) response = %#v", nilPutResp)
	}
}

func TestServerTrimsWorkflowNameBeforeStoring(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()
	doc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {
			"name": " padded-workflow "
		},
		"spec": {}
	}`)

	resp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc})
	if err != nil {
		t.Fatalf("CreateWorkflow() error = %v", err)
	}
	if _, ok := resp.(createWorkflow200Response); !ok {
		t.Fatalf("CreateWorkflow() response = %#v", resp)
	}

	gotResp, err := srv.GetWorkflow(ctx, adminservice.GetWorkflowRequestObject{Name: "padded-workflow"})
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	got, ok := gotResp.(getWorkflow200Response)
	if !ok {
		t.Fatalf("GetWorkflow() response = %#v", gotResp)
	}
	flowcraft := mustFlowcraft(t, got.doc)
	if flowcraft.Metadata.Name != "padded-workflow" {
		t.Fatalf("GetWorkflow() metadata.name = %q, want padded-workflow", flowcraft.Metadata.Name)
	}
}

func TestServerListWorkflowsPagination(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		doc := mustDocument(t, fmt.Sprintf(`{
			"apiVersion": "gizclaw.flowcraft/v1alpha1",
			"kind": "FlowcraftWorkflow",
			"metadata": {
				"name": %q
			},
			"spec": {}
		}`, name))
		if _, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc}); err != nil {
			t.Fatalf("CreateWorkflow(%q) error = %v", name, err)
		}
	}

	limit := adminservice.Limit(1)
	firstResp, err := srv.ListWorkflows(ctx, adminservice.ListWorkflowsRequestObject{
		Params: adminservice.ListWorkflowsParams{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListWorkflows(first page) error = %v", err)
	}
	first, ok := firstResp.(adminservice.ListWorkflows200JSONResponse)
	if !ok {
		t.Fatalf("ListWorkflows(first page) response = %#v", firstResp)
	}
	if len(first.Items) != 1 || !first.HasNext || first.NextCursor == nil {
		t.Fatalf("ListWorkflows(first page) = %#v", first)
	}

	cursor := adminservice.Cursor(*first.NextCursor)
	secondResp, err := srv.ListWorkflows(ctx, adminservice.ListWorkflowsRequestObject{
		Params: adminservice.ListWorkflowsParams{
			Cursor: &cursor,
			Limit:  &limit,
		},
	})
	if err != nil {
		t.Fatalf("ListWorkflows(second page) error = %v", err)
	}
	second, ok := secondResp.(adminservice.ListWorkflows200JSONResponse)
	if !ok {
		t.Fatalf("ListWorkflows(second page) response = %#v", secondResp)
	}
	if len(second.Items) != 1 {
		t.Fatalf("ListWorkflows(second page) = %#v", second)
	}
}

func TestServerWorkflowConflictAndMissingDelete(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()
	doc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {"name": "duplicate"},
		"spec": {}
	}`)
	if _, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc}); err != nil {
		t.Fatalf("CreateWorkflow(seed) error = %v", err)
	}
	duplicateResp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc})
	if err != nil {
		t.Fatalf("CreateWorkflow(duplicate) error = %v", err)
	}
	if _, ok := duplicateResp.(adminservice.CreateWorkflow409JSONResponse); !ok {
		t.Fatalf("CreateWorkflow(duplicate) response = %#v", duplicateResp)
	}

	deleteResp, err := srv.DeleteWorkflow(ctx, adminservice.DeleteWorkflowRequestObject{Name: "missing"})
	if err != nil {
		t.Fatalf("DeleteWorkflow(missing) error = %v", err)
	}
	if _, ok := deleteResp.(adminservice.DeleteWorkflow404JSONResponse); !ok {
		t.Fatalf("DeleteWorkflow(missing) response = %#v", deleteResp)
	}
}

func TestServerWorkflowStoreNotConfigured(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	ctx := context.Background()
	doc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {"name": "missing-store"},
		"spec": {}
	}`)

	listResp, err := srv.ListWorkflows(ctx, adminservice.ListWorkflowsRequestObject{})
	if err != nil {
		t.Fatalf("ListWorkflows() error = %v", err)
	}
	if _, ok := listResp.(adminservice.ListWorkflows500JSONResponse); !ok {
		t.Fatalf("ListWorkflows() response = %#v", listResp)
	}
	createResp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc})
	if err != nil {
		t.Fatalf("CreateWorkflow() error = %v", err)
	}
	if _, ok := createResp.(adminservice.CreateWorkflow500JSONResponse); !ok {
		t.Fatalf("CreateWorkflow() response = %#v", createResp)
	}
	getResp, err := srv.GetWorkflow(ctx, adminservice.GetWorkflowRequestObject{Name: "missing-store"})
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	if _, ok := getResp.(adminservice.GetWorkflow500JSONResponse); !ok {
		t.Fatalf("GetWorkflow() response = %#v", getResp)
	}
}

func TestServerRejectsMissingWorkflowRequiredFields(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()
	for name, raw := range map[string]string{
		"apiVersion": `{"kind":"FlowcraftWorkflow","metadata":{"name":"bad"},"spec":{}}`,
		"kind":       `{"apiVersion":"gizclaw.flowcraft/v1alpha1","metadata":{"name":"bad"},"spec":{}}`,
		"name":       `{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"FlowcraftWorkflow","metadata":{},"spec":{}}`,
		"spec":       `{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"FlowcraftWorkflow","metadata":{"name":"bad"}}`,
	} {
		doc := mustDocument(t, raw)
		resp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc})
		if err != nil {
			t.Fatalf("CreateWorkflow(%s) error = %v", name, err)
		}
		if _, ok := resp.(adminservice.CreateWorkflow400JSONResponse); !ok {
			t.Fatalf("CreateWorkflow(%s) response = %#v", name, resp)
		}
	}
}

func TestServerRejectsUnsupportedWorkflowVersion(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()
	doc := mustDocument(t, `{
		"apiVersion": "example.invalid/v1",
		"kind": "FlowcraftWorkflow",
		"metadata": {"name": "bad-version"},
		"spec": {}
	}`)
	resp, err := srv.CreateWorkflow(ctx, adminservice.CreateWorkflowRequestObject{Body: &doc})
	if err != nil {
		t.Fatalf("CreateWorkflow(bad version) error = %v", err)
	}
	if _, ok := resp.(adminservice.CreateWorkflow400JSONResponse); !ok {
		t.Fatalf("CreateWorkflow(bad version) response = %#v", resp)
	}
}

func TestWorkflowResponseVisitors(t *testing.T) {
	t.Parallel()

	doc := mustDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "FlowcraftWorkflow",
		"metadata": {"name": "visitor"},
		"spec": {}
	}`)
	cases := map[string]func(*fiber.Ctx) error{
		"create": createWorkflow200Response{doc: doc}.VisitCreateWorkflowResponse,
		"get":    getWorkflow200Response{doc: doc}.VisitGetWorkflowResponse,
		"put":    putWorkflow200Response{doc: doc}.VisitPutWorkflowResponse,
		"delete": deleteWorkflow200Response{doc: doc}.VisitDeleteWorkflowResponse,
	}
	for name, visit := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Get("/", visit)
			resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			if got := resp.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("content-type = %q, want application/json", got)
			}
		})
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	store, err := kv.NewBadgerInMemory(nil)
	if err != nil {
		t.Fatalf("NewBadgerInMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &Server{Store: store}
}

func mustDocument(t *testing.T, raw string) apitypes.WorkflowDocument {
	t.Helper()

	var doc apitypes.WorkflowDocument
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return doc
}

func discriminatorOf(t *testing.T, doc apitypes.WorkflowDocument) string {
	t.Helper()

	return string(doc.Kind)
}

func mustSingle(t *testing.T, doc apitypes.WorkflowDocument) apitypes.FlowcraftWorkflow {
	t.Helper()

	return mustFlowcraft(t, doc)
}

func mustFlowcraft(t *testing.T, doc apitypes.WorkflowDocument) apitypes.FlowcraftWorkflow {
	t.Helper()

	return doc
}
