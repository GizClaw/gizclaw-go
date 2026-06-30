package gizclaw

import (
	"context"
	"net"

	"github.com/GizClaw/gizclaw-go/pkgs/gizclaw/api/rpcapi"
)

func (c *rpcClient) GetClientInfo(ctx context.Context, conn net.Conn, id string) (*rpcapi.ClientGetInfoResponse, error) {
	params, err := newRPCRequestParams(rpcapi.ClientGetInfoRequest{}, (*rpcapi.RPCRequest_Params).FromClientGetInfoRequest)
	if err != nil {
		return nil, err
	}
	result, err := callRPCResult(ctx, conn, newRPCRequest(id, rpcapi.RPCMethodClientInfoGet, params), rpcapi.RPCResponse_Result.AsClientGetInfoResponse)
	if err != nil {
		return nil, wrapRPCResultError("device info", err)
	}
	return result, nil
}

func (c *rpcClient) GetClientIdentifiers(ctx context.Context, conn net.Conn, id string) (*rpcapi.ClientGetIdentifiersResponse, error) {
	params, err := newRPCRequestParams(rpcapi.ClientGetIdentifiersRequest{}, (*rpcapi.RPCRequest_Params).FromClientGetIdentifiersRequest)
	if err != nil {
		return nil, err
	}
	result, err := callRPCResult(ctx, conn, newRPCRequest(id, rpcapi.RPCMethodClientIdentifiersGet, params), rpcapi.RPCResponse_Result.AsClientGetIdentifiersResponse)
	if err != nil {
		return nil, wrapRPCResultError("device identifiers", err)
	}
	return result, nil
}
