package gizclaw

import (
	"context"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
)

type ServerSecurityPolicy Server

var _ giznet.SecurityPolicy = (*ServerSecurityPolicy)(nil)

func (p *ServerSecurityPolicy) AllowPeer(giznet.PublicKey) bool {
	return p != nil
}

func (p *ServerSecurityPolicy) AllowService(publicKey giznet.PublicKey, service uint64) bool {
	if p == nil {
		return false
	}
	s := (*Server)(p)
	if m := s.manager; m != nil && m.allowService(context.Background(), publicKey, service) {
		return true
	}
	return s.SecurityPolicy != nil && s.SecurityPolicy.AllowService(publicKey, service)
}
