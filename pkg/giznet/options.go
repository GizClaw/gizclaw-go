package giznet

import "github.com/GizClaw/gizclaw-go/pkg/giznet/internal/core"

type Option = core.Option
type SocketConfig = core.SocketConfig
type ServiceMuxConfig = core.ServiceMuxConfig

var (
	WithBindAddr          = core.WithBindAddr
	WithAllowUnknown      = core.WithAllowUnknown
	WithDecryptWorkers    = core.WithDecryptWorkers
	WithRawChanSize       = core.WithRawChanSize
	WithDecryptedChanSize = core.WithDecryptedChanSize
	WithSocketConfig      = core.WithSocketConfig
	WithServiceMuxConfig  = core.WithServiceMuxConfig
	WithOnPeerEvent       = core.WithOnPeerEvent
)
