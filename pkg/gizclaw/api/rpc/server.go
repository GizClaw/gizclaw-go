package rpc

import (
	"context"
	"time"
)

type RPCServer struct {
}

var _ StrictServerInterface = (*RPCServer)(nil)

func (s *RPCServer) Ping(_ context.Context, _ PingRequest) (*PingResponse, error) {
	return &PingResponse{ServerTime: time.Now().UnixMilli()}, nil
}
