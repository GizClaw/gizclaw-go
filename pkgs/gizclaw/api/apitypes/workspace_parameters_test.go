package apitypes

import (
	"encoding/json"
	"testing"
)

func TestWorkspaceParametersChatRoomBranch(t *testing.T) {
	mode := ChatRoomModeGroup
	input := WorkspaceInputModePushToTalk
	ttl := "168h"
	asr := "e2e-asr"
	transcriptEnabled := true
	var params WorkspaceParameters
	if err := params.FromChatRoomWorkspaceParameters(ChatRoomWorkspaceParameters{
		Input: &input,
		Mode:  &mode,
		History: &ChatRoomWorkspaceHistoryParameters{
			Ttl: &ttl,
		},
		Transcript: &ChatRoomWorkspaceTranscriptParameters{
			Enabled:  &transcriptEnabled,
			AsrModel: &asr,
		},
	}); err != nil {
		t.Fatalf("FromChatRoomWorkspaceParameters() error = %v", err)
	}
	if got, err := params.Discriminator(); err != nil || got != "chatroom" {
		t.Fatalf("Discriminator() = %q, %v; want chatroom", got, err)
	}
	value, err := params.ValueByDiscriminator()
	if err != nil {
		t.Fatalf("ValueByDiscriminator() error = %v", err)
	}
	typed, ok := value.(ChatRoomWorkspaceParameters)
	if !ok {
		t.Fatalf("ValueByDiscriminator() = %T, want ChatRoomWorkspaceParameters", value)
	}
	if typed.AgentType != ChatRoomWorkspaceParametersAgentTypeChatroom {
		t.Fatalf("agent_type = %q", typed.AgentType)
	}
	if typed.History == nil || typed.History.Ttl == nil || *typed.History.Ttl != ttl {
		t.Fatalf("history = %#v", typed.History)
	}
	if typed.Transcript == nil || typed.Transcript.AsrModel == nil || *typed.Transcript.AsrModel != asr {
		t.Fatalf("transcript = %#v", typed.Transcript)
	}
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("MarshalJSON() produced invalid JSON: %s", raw)
	}
}
