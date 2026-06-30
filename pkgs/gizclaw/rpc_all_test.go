package gizclaw

import (
	"context"
	"net"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/gizclaw/api/rpcapi"
)

func TestRPCPeerPingServerHandle(t *testing.T) {
	server := &rpcServer{}
	ping := callRPCPair(t, server, func(conn net.Conn) (*rpcapi.PingResponse, error) {
		return callRPCPing(context.Background(), conn, "ping")
	})
	if ping.ServerTime <= 0 {
		t.Fatalf("Ping() = %+v", ping)
	}
}
