package gizclaw

import (
	"context"
	"net"

	"github.com/GizClaw/gizclaw-go/pkgs/gizclaw/api/rpcapi"
)

func (s *rpcServer) Ping(ctx context.Context, conn net.Conn, id string) (*rpcapi.PingResponse, error) {
	return callRPCPing(ctx, conn, id)
}
