package apitypes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatRoomWorkflowSpecJSON(t *testing.T) {
	raw := []byte(`{
		"history": {
			"ttl": "168h"
		},
		"transcript": {
			"enabled": true,
			"asr_model": "e2e-asr"
		}
	}`)
	var spec ChatRoomWorkflowSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("json.Unmarshal(ChatRoomWorkflowSpec) error = %v", err)
	}
	if spec.History.Ttl == nil || *spec.History.Ttl != "168h" {
		t.Fatalf("history = %#v", spec.History)
	}
}

func TestChatRoomWorkflowSpecRejectsInvalidJSON(t *testing.T) {
	for name, tc := range map[string]struct {
		raw     string
		wantErr string
	}{
		"missing history": {
			raw:     `{}`,
			wantErr: "history is required",
		},
		"null history": {
			raw:     `{"history": null}`,
			wantErr: "history is required",
		},
		"history enabled": {
			raw: `{
				"history": {
					"enabled": true,
					"ttl": "168h"
				}
			}`,
			wantErr: "unknown field",
		},
		"delivery": {
			raw: `{
				"history": {
					"ttl": "168h"
				},
				"delivery": {
					"default_route": "broadcast"
				}
			}`,
			wantErr: "unknown field",
		},
	} {
		var spec ChatRoomWorkflowSpec
		err := json.Unmarshal([]byte(tc.raw), &spec)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("%s: json.Unmarshal() error = %v, want %q", name, err, tc.wantErr)
		}
	}
}
