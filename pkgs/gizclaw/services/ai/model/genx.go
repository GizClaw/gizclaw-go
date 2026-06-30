package model

import (
	"github.com/GizClaw/gizclaw-go/pkgs/genx"
	"github.com/GizClaw/gizclaw-go/pkgs/gizclaw/services/ai/peergenx"
)

// NewGenerator returns a server-owned GenX generator for model/<id> patterns.
func NewGenerator(service peergenx.Service) genx.Generator {
	return peergenx.New(service).Generator()
}
