package voice

import (
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/genx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/peergenx"
)

func TestNewTransformerReturnsGenXTransformer(t *testing.T) {
	var got genx.Transformer = NewTransformer(peergenx.Service{})
	if got == nil {
		t.Fatal("NewTransformer() = nil")
	}
}
