package rpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestClientPingSingleRequestResponse(t *testing.T) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	client := NewClient(clientSide)
	defer client.Close()

	reqCh := make(chan *RPCRequest, 1)
	serverErrCh := make(chan error, 1)

	go func() {
		req, err := ReadRequest(serverSide)
		if err != nil {
			serverErrCh <- err
			return
		}
		reqCh <- req

		resp := ResultResponse(req.Id, &PingResponse{ServerTime: serverTimeForID(req.Id)})
		if err := WriteResponse(serverSide, resp); err != nil {
			serverErrCh <- err
			return
		}

		serverErrCh <- nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ping, err := client.Ping(ctx, "req-1")
	if err != nil {
		t.Fatalf("Ping(req-1) error: %v", err)
	}
	if ping.ServerTime != serverTimeForID("req-1") {
		t.Fatalf("Ping(req-1) server_time = %d", ping.ServerTime)
	}

	if req := <-reqCh; req.Id == "" {
		t.Fatal("request missing id")
	} else {
		assertPingRequestHasTimestamp(t, req)
	}
	if err := <-serverErrCh; err != nil {
		t.Fatalf("server goroutine error: %v", err)
	}
}

func TestClientCallContextTimeout(t *testing.T) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	client := NewClient(clientSide)
	defer client.Close()

	readDone := make(chan *RPCRequest, 1)
	go func() {
		req, _ := ReadRequest(serverSide)
		readDone <- req
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.call(ctx, &RPCRequest{
		V:      1,
		Id:     "timeout",
		Method: MethodPing,
		Params: &PingRequest{ClientSendTime: time.Now().UnixMilli()},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Call timeout err = %v, want %v", err, context.DeadlineExceeded)
	}

	if req := <-readDone; req == nil || req.Id != "timeout" {
		t.Fatalf("server received request = %+v", req)
	} else {
		assertPingRequestHasTimestamp(t, req)
	}
}

func TestClientCallContextCancel(t *testing.T) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	client := NewClient(clientSide)
	defer client.Close()

	readDone := make(chan *RPCRequest, 1)
	go func() {
		req, _ := ReadRequest(serverSide)
		readDone <- req
	}()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if req := <-readDone; req != nil {
			cancel()
		}
	}()

	_, err := client.call(ctx, &RPCRequest{
		V:      1,
		Id:     "cancel",
		Method: MethodPing,
		Params: &PingRequest{ClientSendTime: time.Now().UnixMilli()},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Call cancel err = %v, want %v", err, context.Canceled)
	}
}

func TestClientCallValidatesRequestAndClosedState(t *testing.T) {
	client := NewClient(&nopReadWriter{})
	if _, err := client.call(context.Background(), nil); err == nil || err.Error() != "rpc: nil request" {
		t.Fatalf("call(nil) err = %v", err)
	}
	if _, err := client.call(context.Background(), &RPCRequest{Method: MethodPing}); err == nil || err.Error() != "rpc: request id required" {
		t.Fatalf("call(empty id) err = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := client.call(context.Background(), &RPCRequest{Id: "closed", Method: MethodPing}); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("call(closed) err = %v, want %v", err, ErrClientClosed)
	}
}

func TestClientPingErrorPaths(t *testing.T) {
	t.Run("error response", func(t *testing.T) {
		serverSide, clientSide := net.Pipe()
		defer serverSide.Close()
		defer clientSide.Close()

		client := NewClient(clientSide)
		defer client.Close()

		go func() {
			req, _ := ReadRequest(serverSide)
			_ = WriteResponse(serverSide, ErrorResponse(req.Id, -1, "boom"))
		}()

		_, err := client.Ping(context.Background(), "ping-error")
		if err == nil || err.Error() != "rpc: boom" {
			t.Fatalf("Ping(error response) err = %v", err)
		}
	})

	t.Run("missing result", func(t *testing.T) {
		serverSide, clientSide := net.Pipe()
		defer serverSide.Close()
		defer clientSide.Close()

		client := NewClient(clientSide)
		defer client.Close()

		go func() {
			req, _ := ReadRequest(serverSide)
			_ = WriteResponse(serverSide, &RPCResponse{V: 1, Id: req.Id})
		}()

		_, err := client.Ping(context.Background(), "ping-missing")
		if err == nil || err.Error() != "rpc: missing ping result" {
			t.Fatalf("Ping(missing result) err = %v", err)
		}
	})
}

func TestServerDispatchesStrictServerInterface(t *testing.T) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	impl := &testRPCServer{
		resp: &PingResponse{ServerTime: 456},
	}
	server := NewServer(impl)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ServeContext(context.WithValue(context.Background(), rpcTestContextKey{}, "ctx-value"), serverSide)
	}()

	if err := WriteRequest(clientSide, &RPCRequest{
		V:      1,
		Id:     "ping-1",
		Method: MethodPing,
		Params: &PingRequest{ClientSendTime: 123},
	}); err != nil {
		t.Fatalf("WriteRequest error: %v", err)
	}

	resp, err := ReadResponse(clientSide)
	if err != nil {
		t.Fatalf("ReadResponse error: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("response error = %+v", resp.Error)
	}
	if resp.Result == nil || resp.Result.ServerTime != 456 {
		t.Fatalf("response result = %+v", resp.Result)
	}
	if impl.got.ClientSendTime != 123 {
		t.Fatalf("Ping request client_send_time = %d", impl.got.ClientSendTime)
	}
	if got := impl.ctx.Value(rpcTestContextKey{}); got != "ctx-value" {
		t.Fatalf("context value = %v", got)
	}
	if err := <-serverErrCh; err != nil {
		t.Fatalf("ServeContext error: %v", err)
	}
}

func TestServerServeContextRequiresImplementation(t *testing.T) {
	var server *Server
	if err := server.ServeContext(context.Background(), &nopReadWriter{}); err == nil || err.Error() != "rpc: nil server implementation" {
		t.Fatalf("ServeContext(nil server) err = %v", err)
	}

	server = &Server{}
	if err := server.ServeContext(context.Background(), &nopReadWriter{}); err == nil || err.Error() != "rpc: nil server implementation" {
		t.Fatalf("ServeContext(nil impl) err = %v", err)
	}
}

func TestServerServeContextNormalizesNilResponse(t *testing.T) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	server := NewServer(nilResponseRPCServer{})
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ServeContext(context.Background(), serverSide)
	}()

	if err := WriteRequest(clientSide, &RPCRequest{
		V:      1,
		Id:     "ping-normalize",
		Method: MethodPing,
		Params: &PingRequest{ClientSendTime: 1},
	}); err != nil {
		t.Fatalf("WriteRequest error = %v", err)
	}

	resp, err := ReadResponse(clientSide)
	if err != nil {
		t.Fatalf("ReadResponse error = %v", err)
	}
	if resp == nil || resp.Id != "ping-normalize" || resp.V != 1 {
		t.Fatalf("normalized response = %+v", resp)
	}
	if err := <-serverErrCh; err != nil {
		t.Fatalf("ServeContext error: %v", err)
	}
}

func TestServerDispatchUnknownMethod(t *testing.T) {
	server := NewServer(&testRPCServer{resp: &PingResponse{ServerTime: 1}})
	resp, err := server.dispatch(context.Background(), &RPCRequest{
		Id:     "unknown",
		Method: "rpc.unknown",
	})
	if err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}
	if resp == nil || resp.Error == nil || resp.Error.Message != "unknown method: rpc.unknown" {
		t.Fatalf("dispatch() response = %+v", resp)
	}
}

func TestReadRequestAndResponseRejectInvalidJSON(t *testing.T) {
	var reqBuf bytes.Buffer
	if err := WriteFrame(&reqBuf, []byte("{")); err != nil {
		t.Fatalf("WriteFrame(request) error = %v", err)
	}
	if _, err := ReadRequest(&reqBuf); err == nil {
		t.Fatal("ReadRequest() should fail for invalid JSON")
	}

	var respBuf bytes.Buffer
	if err := WriteFrame(&respBuf, []byte("{")); err != nil {
		t.Fatalf("WriteFrame(response) error = %v", err)
	}
	if _, err := ReadResponse(&respBuf); err == nil {
		t.Fatal("ReadResponse() should fail for invalid JSON")
	}
}

func TestServerReturnsErrorForMissingPingParams(t *testing.T) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	server := NewServer(&testRPCServer{resp: &PingResponse{ServerTime: 1}})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Serve(clientSide)
	}()

	if err := WriteRequest(serverSide, &RPCRequest{
		V:      1,
		Id:     "ping-1",
		Method: MethodPing,
	}); err != nil {
		t.Fatalf("WriteRequest error: %v", err)
	}

	resp, err := ReadResponse(serverSide)
	if err != nil {
		t.Fatalf("ReadResponse error: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Code != -32602 {
		t.Fatalf("error code = %d", resp.Error.Code)
	}
	if err := <-serverErrCh; err != nil {
		t.Fatalf("Serve error: %v", err)
	}
}

func serverTimeForID(id string) int64 {
	if id == "req-1" {
		return 1
	}
	return 2
}

func assertPingRequestHasTimestamp(t *testing.T, req *RPCRequest) {
	t.Helper()
	if req.Params == nil {
		t.Fatal("ping request params missing")
	}
	if req.Params.ClientSendTime <= 0 {
		t.Fatalf("ping request client_send_time = %d", req.Params.ClientSendTime)
	}
}

type testRPCServer struct {
	ctx  context.Context
	got  PingRequest
	resp *PingResponse
}

func (s *testRPCServer) Ping(ctx context.Context, request PingRequest) (*PingResponse, error) {
	s.ctx = ctx
	s.got = request
	return s.resp, nil
}

type rpcTestContextKey struct{}

type nopReadWriter struct{}

func (*nopReadWriter) Read(_ []byte) (int, error)  { return 0, io.EOF }
func (*nopReadWriter) Write(p []byte) (int, error) { return len(p), nil }

type nilResponseRPCServer struct{}

func (nilResponseRPCServer) Ping(context.Context, PingRequest) (*PingResponse, error) {
	return nil, nil
}
