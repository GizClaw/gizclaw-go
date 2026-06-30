package apitypes

import (
	"bytes"
	"encoding/json"
	"errors"
)

func (s *ChatRoomWorkflowSpec) UnmarshalJSON(data []byte) error {
	type chatRoomWorkflowSpec ChatRoomWorkflowSpec
	var decoded chatRoomWorkflowSpec
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&decoded); err != nil {
		return err
	}
	var required struct {
		History *json.RawMessage `json:"history"`
	}
	if err := json.Unmarshal(data, &required); err != nil {
		return err
	}
	if required.History == nil || bytes.Equal(bytes.TrimSpace(*required.History), []byte("null")) {
		return errors.New("history is required")
	}
	*s = ChatRoomWorkflowSpec(decoded)
	return nil
}
