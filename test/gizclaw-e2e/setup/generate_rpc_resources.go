//go:build ignore

package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	apiVersion = "gizclaw.admin/v1alpha1"

	e2eClientPublicKey = "8rAUkTyxLHDa5o3VajtzWcQdNJq1thrjAGtpwQkEsaEu"

	workflowCount   = 120
	workspaceCount  = 120
	modelCount      = 80
	credentialCount = 50
	firmwareCount   = 80
)

type resource = map[string]any

func main() {
	root := filepath.Join("test", "gizclaw-e2e", "testdata", "resources")
	must(os.MkdirAll(filepath.Join(root, "assets", "firmware"), 0o755))

	writeResourceList(root, "070-rpc-core.json", "rpc-core", coreResources())
	writeResourceList(root, "071-rpc-catalog-workflows.json", "rpc-catalog-workflows", workflowCatalog())
	writeResourceList(root, "072-rpc-catalog-workspaces.json", "rpc-catalog-workspaces", workspaceCatalog())
	writeResourceList(root, "073-rpc-catalog-models.json", "rpc-catalog-models", modelCatalog())
	writeResourceList(root, "074-rpc-catalog-credentials.json", "rpc-catalog-credentials", credentialCatalog())
	writeResourceList(root, "075-rpc-catalog-firmwares.json", "rpc-catalog-firmwares", firmwareCatalog())
	writeResource(root, "076-rpc-acl-role.json", aclRoleResource("e2e-rpc-client", []string{
		"credential.admin",
		"credential.read",
		"credential.use",
		"firmware.read",
		"model.admin",
		"model.read",
		"model.use",
		"voice.read",
		"voice.use",
		"workflow.admin",
		"workflow.read",
		"workflow.use",
		"workspace.admin",
		"workspace.read",
		"workspace.use",
	}))
	writeResourceList(root, "077-rpc-acl-bindings.json", "rpc-acl-bindings", aclBindings())
	writeResourceList(root, "078-rpc-mutation-fixtures.json", "rpc-mutation-fixtures", mutationBindings())
	writeResourceList(root, "079-rpc-history-workspace.json", "rpc-history-workspace", historyResources())
	writeFirmwareTar(filepath.Join(root, "assets", "firmware", "e2e-rpc-firmware-main.tar"))
}

func writeResourceList(root, name, listName string, items []resource) {
	writeResource(root, name, resource{
		"apiVersion": apiVersion,
		"kind":       "ResourceList",
		"metadata": resource{
			"name": listName,
		},
		"spec": resource{
			"items": items,
		},
	})
}

func writeResource(root, name string, value resource) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	must(encoder.Encode(value))
	must(os.WriteFile(filepath.Join(root, name), buf.Bytes(), 0o644))
}

func coreResources() []resource {
	return []resource{
		credentialResource("e2e-rpc-credential", "sk-e2e-rpc"),
		modelResource("e2e-rpc-model", "e2e-rpc-provider", "e2e-rpc-core-model"),
		voiceResource("e2e-rpc-voice", "e2e-rpc-provider", "e2e-rpc-voice-id"),
		workflowResource("e2e-rpc-workflow", "Stable RPC workflow fixture"),
		chatroomWorkflowResource("e2e-rpc-run-workflow", "Run-control RPC workflow fixture"),
		workspaceResource("e2e-rpc-workspace", "e2e-rpc-workflow"),
		chatroomWorkspaceResource("e2e-rpc-run-workspace", "e2e-rpc-run-workflow"),
		firmwareResource("e2e-rpc-firmware", "9.9.0", true),
	}
}

func workflowCatalog() []resource {
	items := make([]resource, 0, workflowCount)
	for i := 0; i < workflowCount; i++ {
		items = append(items, workflowResource(fmt.Sprintf("e2e-rpc-workflow-%03d", i), fmt.Sprintf("Bulk RPC workflow fixture %03d", i)))
	}
	return items
}

func workspaceCatalog() []resource {
	items := make([]resource, 0, workspaceCount)
	for i := 0; i < workspaceCount; i++ {
		items = append(items, workspaceResource(fmt.Sprintf("e2e-rpc-workspace-%03d", i), fmt.Sprintf("e2e-rpc-workflow-%03d", i%workflowCount)))
	}
	return items
}

func modelCatalog() []resource {
	items := make([]resource, 0, modelCount)
	for i := 0; i < modelCount; i++ {
		items = append(items, modelResource(fmt.Sprintf("e2e-rpc-model-%03d", i), "e2e-rpc-provider", fmt.Sprintf("e2e-rpc-upstream-%03d", i)))
	}
	return items
}

func credentialCatalog() []resource {
	items := make([]resource, 0, credentialCount)
	for i := 0; i < credentialCount; i++ {
		items = append(items, credentialResource(fmt.Sprintf("e2e-rpc-credential-%03d", i), fmt.Sprintf("sk-e2e-rpc-%03d", i)))
	}
	return items
}

func firmwareCatalog() []resource {
	items := make([]resource, 0, firmwareCount)
	for i := 0; i < firmwareCount; i++ {
		items = append(items, firmwareResource(fmt.Sprintf("e2e-rpc-firmware-%03d", i), fmt.Sprintf("1.%d.%d", i/10, i%10), false))
	}
	return items
}

func historyResources() []resource {
	return []resource{
		chatroomWorkflowResource("e2e-rpc-history-workflow", "History/replay RPC workflow fixture"),
		chatroomWorkspaceResource("e2e-rpc-history-workspace", "e2e-rpc-history-workflow"),
	}
}

func credentialResource(name, apiKey string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Credential",
		"metadata": resource{
			"name": name,
		},
		"spec": resource{
			"provider":    "openai",
			"description": "Schema-valid fake RPC e2e credential",
			"body": resource{
				"api_key": apiKey,
			},
		},
	}
}

func modelResource(id, provider, upstream string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Model",
		"metadata": resource{
			"name": id,
		},
		"spec": resource{
			"kind":   "llm",
			"source": "manual",
			"name":   fmt.Sprintf("RPC Fixture %s", id),
			"provider": resource{
				"kind": "openai-tenant",
				"name": provider,
			},
			"provider_data": resource{
				"upstream_model":      upstream,
				"support_json_output": true,
				"use_system_role":     true,
			},
		},
	}
}

func voiceResource(id, provider, voiceID string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Voice",
		"metadata": resource{
			"name": id,
		},
		"spec": resource{
			"source": "manual",
			"provider": resource{
				"kind": "minimax-tenant",
				"name": provider,
			},
			"name":        "RPC Fixture Voice",
			"description": "Manual fake voice row for RPC e2e metadata coverage",
			"provider_data": resource{
				"voice_id":   voiceID,
				"voice_type": "voice_cloning",
				"raw": resource{
					"origin": "rpc-e2e-fixture",
				},
			},
		},
	}
}

func workflowResource(name, description string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Workflow",
		"metadata": resource{
			"name":        name,
			"description": description,
		},
		"spec": resource{
			"driver": "flowcraft",
			"flowcraft": resource{
				"entry_agent": "",
			},
		},
	}
}

func chatroomWorkflowResource(name, description string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Workflow",
		"metadata": resource{
			"name":        name,
			"description": description,
		},
		"spec": resource{
			"driver": "chatroom",
			"chatroom": resource{
				"history": resource{
					"ttl": "168h",
				},
			},
		},
	}
}

func workspaceResource(name, workflowName string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Workspace",
		"metadata": resource{
			"name": name,
		},
		"spec": resource{
			"workflow_name": workflowName,
			"parameters": resource{
				"agent_type": "flowcraft",
				"input":      "push-to-talk",
			},
		},
	}
}

func chatroomWorkspaceResource(name, workflowName string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Workspace",
		"metadata": resource{
			"name": name,
		},
		"spec": resource{
			"workflow_name": workflowName,
			"parameters": resource{
				"agent_type": "chatroom",
			},
		},
	}
}

func firmwareResource(name, version string, withArtifact bool) resource {
	stable := resource{"version": version}
	if withArtifact {
		stable["artifacts"] = []resource{{
			"name": "main",
			"kind": "app",
		}}
	}
	return resource{
		"apiVersion": apiVersion,
		"kind":       "Firmware",
		"metadata": resource{
			"name": name,
		},
		"spec": resource{
			"description": "RPC e2e firmware fixture",
			"slots": resource{
				"stable":  stable,
				"beta":    resource{"version": version + "-beta"},
				"develop": resource{"version": version + "-dev"},
			},
		},
	}
}

func aclRoleResource(name string, permissions []string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "ACLRole",
		"metadata": resource{
			"name": name,
		},
		"spec": resource{
			"permissions": permissions,
		},
	}
}

func aclBindings() []resource {
	items := []resource{
		aclBinding("view-e2e-client-rpc-workflow-collection", "workflow", "__collection__", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-workspace-collection", "workspace", "__collection__", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-model-core", "model", "e2e-rpc-model", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-credential-core", "credential", "e2e-rpc-credential", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-firmware-core", "firmware", "e2e-rpc-firmware", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-voice-core", "voice", "e2e-rpc-voice", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-history-workspace", "workspace", "e2e-rpc-history-workspace", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-history-workflow", "workflow", "e2e-rpc-history-workflow", "e2e-rpc-client"),
		aclPublicKeyBinding("pk-e2e-client-rpc-history-workspace", e2eClientPublicKey, "workspace", "e2e-rpc-history-workspace", "e2e-rpc-client"),
	}
	for i := 0; i < modelCount; i++ {
		name := fmt.Sprintf("e2e-rpc-model-%03d", i)
		items = append(items, aclBinding("view-e2e-client-rpc-model-"+fmt.Sprintf("%03d", i), "model", name, "e2e-rpc-client"))
	}
	for i := 0; i < credentialCount; i++ {
		name := fmt.Sprintf("e2e-rpc-credential-%03d", i)
		items = append(items, aclBinding("view-e2e-client-rpc-credential-"+fmt.Sprintf("%03d", i), "credential", name, "e2e-rpc-client"))
	}
	for i := 0; i < firmwareCount; i++ {
		name := fmt.Sprintf("e2e-rpc-firmware-%03d", i)
		items = append(items, aclBinding("view-e2e-client-rpc-firmware-"+fmt.Sprintf("%03d", i), "firmware", name, "e2e-rpc-client"))
	}
	return items
}

func mutationBindings() []resource {
	return []resource{
		aclBinding("view-e2e-client-rpc-mut-model", "model", "e2e-rpc-mut-model", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-mut-credential", "credential", "e2e-rpc-mut-credential", "e2e-rpc-client"),
		aclBinding("view-e2e-client-rpc-mut-firmware", "firmware", "e2e-rpc-mut-firmware", "e2e-rpc-client"),
	}
}

func aclBinding(name, kind, id, role string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "ACLPolicyBinding",
		"metadata": resource{
			"name": name,
		},
		"spec": resource{
			"subject": resource{
				"kind": "view",
				"id":   "e2e-client",
			},
			"resource": resource{
				"kind": kind,
				"id":   id,
			},
			"role": role,
		},
	}
}

func aclPublicKeyBinding(name, peerPublicKey, kind, id, role string) resource {
	return resource{
		"apiVersion": apiVersion,
		"kind":       "ACLPolicyBinding",
		"metadata": resource{
			"name": name,
		},
		"spec": resource{
			"subject": resource{
				"kind": "pk",
				"id":   peerPublicKey,
			},
			"resource": resource{
				"kind": kind,
				"id":   id,
			},
			"role": role,
		},
	}
}

func writeFirmwareTar(path string) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	files := []struct {
		name string
		body string
	}{
		{name: "MANIFEST.txt", body: "gizclaw e2e rpc firmware\nversion: 9.9.0\nartifact: main\n"},
		{name: "payload.bin", body: "deterministic rpc firmware payload\n"},
	}
	modTime := time.Unix(1700000000, 0).UTC()
	for _, file := range files {
		header := &tar.Header{
			Name:    file.name,
			Mode:    0o644,
			Size:    int64(len(file.body)),
			ModTime: modTime,
		}
		must(tw.WriteHeader(header))
		_, err := tw.Write([]byte(file.body))
		must(err)
	}
	must(tw.Close())
	must(os.WriteFile(path, buf.Bytes(), 0o644))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
