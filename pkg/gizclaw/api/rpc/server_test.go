package rpc

import (
	"context"
	"testing"
)

func TestRPCServerPing(t *testing.T) {
	server := &RPCServer{}
	resp, err := server.Ping(context.Background(), PingRequest{})
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if resp == nil || resp.ServerTime <= 0 {
		t.Fatalf("Ping() response = %+v", resp)
	}
}
