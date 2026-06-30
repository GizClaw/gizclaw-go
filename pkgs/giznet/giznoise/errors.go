package giznoise

import (
	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/internal/core"
)

var (
	ErrNilListener = giznet.ErrNilListener
	ErrNilConn     = giznet.ErrNilConn
	ErrClosed      = giznet.ErrClosed
	ErrConnClosed  = giznet.ErrConnClosed

	ErrNoSession         = core.ErrNoSession
	ErrPeerNotFound      = core.ErrPeerNotFound
	ErrUDPClosed         = core.ErrClosed
	ErrAcceptQueueClosed = core.ErrAcceptQueueClosed
	ErrKCPMustUseStream  = core.ErrKCPMustUseStream
	ErrServiceMuxClosed  = core.ErrServiceMuxClosed
)
