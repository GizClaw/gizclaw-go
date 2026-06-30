package model

import (
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/genx"
	"github.com/GizClaw/gizclaw-go/pkgs/gizclaw/services/ai/peergenx"
)

func TestNewGeneratorReturnsGenXGenerator(t *testing.T) {
	var got genx.Generator = NewGenerator(peergenx.Service{})
	if got == nil {
		t.Fatal("NewGenerator() = nil")
	}
}
