package rpcapi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"
)

func TestFrameRequestResponseRoundTrip(t *testing.T) {
	var frameBuf bytes.Buffer
	if err := WriteFrame(&frameBuf, Frame{Type: FrameTypeBinary, Payload: []byte("payload")}); err != nil {
		t.Fatalf("WriteFrame() error = %v", err)
	}
	frame, err := ReadFrame(&frameBuf)
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	if frame.Type != FrameTypeBinary || string(frame.Payload) != "payload" {
		t.Fatalf("ReadFrame() = %+v", frame)
	}

	var reqParams RPCRequest_Params
	if err := reqParams.FromPingRequest(PingRequest{ClientSendTime: 123}); err != nil {
		t.Fatalf("FromPingRequest() error = %v", err)
	}
	var reqBuf bytes.Buffer
	req := &RPCRequest{
		V:      RPCVersionV1,
		Id:     "req-1",
		Method: RPCMethodPeerPing,
		Params: &reqParams,
	}
	if err := WriteRequest(&reqBuf, req); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}
	gotReq, err := ReadRequest(&reqBuf)
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
	if gotReq.Id != req.Id || gotReq.Method != RPCMethodPeerPing || gotReq.Params == nil {
		t.Fatalf("ReadRequest() = %+v", gotReq)
	}
	gotReqParams, err := gotReq.Params.AsPingRequest()
	if err != nil {
		t.Fatalf("AsPingRequest() error = %v", err)
	}
	if gotReqParams.ClientSendTime != 123 {
		t.Fatalf("AsPingRequest().ClientSendTime = %d", gotReqParams.ClientSendTime)
	}

	var respResult RPCResponse_Result
	if err := respResult.FromPingResponse(PingResponse{ServerTime: 456}); err != nil {
		t.Fatalf("FromPingResponse() error = %v", err)
	}
	var respBuf bytes.Buffer
	resp := &RPCResponse{
		V:      RPCVersionV1,
		Id:     "req-1",
		Result: &respResult,
	}
	if err := WriteResponse(&respBuf, resp); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}
	gotResp, err := ReadResponse(&respBuf)
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
	if gotResp.Id != resp.Id || gotResp.Result == nil {
		t.Fatalf("ReadResponse() = %+v", gotResp)
	}
	gotRespResult, err := gotResp.Result.AsPingResponse()
	if err != nil {
		t.Fatalf("AsPingResponse() error = %v", err)
	}
	if gotRespResult.ServerTime != 456 {
		t.Fatalf("AsPingResponse().ServerTime = %d", gotRespResult.ServerTime)
	}
}

func TestRPCUnionTypes(t *testing.T) {
	var pingParams RPCRequest_Params
	if err := pingParams.MergePingRequest(PingRequest{ClientSendTime: 100}); err != nil {
		t.Fatalf("MergePingRequest() error = %v", err)
	}
	if got, err := pingParams.AsPingRequest(); err != nil || got.ClientSendTime != 100 {
		t.Fatalf("AsPingRequest() = %+v, %v", got, err)
	}

	assertRequestUnion(t, "PeerGetInfo", PeerGetInfoRequest{}, (*RPCRequest_Params).FromPeerGetInfoRequest, RPCRequest_Params.AsPeerGetInfoRequest, (*RPCRequest_Params).MergePeerGetInfoRequest)
	assertRequestUnion(t, "PeerPutInfo", PeerPutInfoRequest{Name: stringPtr("gear-1")}, (*RPCRequest_Params).FromPeerPutInfoRequest, RPCRequest_Params.AsPeerPutInfoRequest, (*RPCRequest_Params).MergePeerPutInfoRequest)
	assertRequestUnion(t, "PeerGetRuntime", PeerGetRuntimeRequest{}, (*RPCRequest_Params).FromPeerGetRuntimeRequest, RPCRequest_Params.AsPeerGetRuntimeRequest, (*RPCRequest_Params).MergePeerGetRuntimeRequest)
	assertRequestUnion(t, "DeviceGetInfo", DeviceGetInfoRequest{}, (*RPCRequest_Params).FromDeviceGetInfoRequest, RPCRequest_Params.AsDeviceGetInfoRequest, (*RPCRequest_Params).MergeDeviceGetInfoRequest)
	assertRequestUnion(t, "DeviceGetIdentifiers", DeviceGetIdentifiersRequest{}, (*RPCRequest_Params).FromDeviceGetIdentifiersRequest, RPCRequest_Params.AsDeviceGetIdentifiersRequest, (*RPCRequest_Params).MergeDeviceGetIdentifiersRequest)
	assertRequestUnion(t, "ServerGetInfo", ServerGetInfoRequest{}, (*RPCRequest_Params).FromServerGetInfoRequest, RPCRequest_Params.AsServerGetInfoRequest, (*RPCRequest_Params).MergeServerGetInfoRequest)

	var pingResult RPCResponse_Result
	if err := pingResult.MergePingResponse(PingResponse{ServerTime: 200}); err != nil {
		t.Fatalf("MergePingResponse() error = %v", err)
	}
	if got, err := pingResult.AsPingResponse(); err != nil || got.ServerTime != 200 {
		t.Fatalf("AsPingResponse() = %+v, %v", got, err)
	}

	now := time.Unix(100, 0).UTC()
	assertResponseUnion(t, "PeerGetInfo", PeerGetInfoResponse{Name: stringPtr("gear-1")}, (*RPCResponse_Result).FromPeerGetInfoResponse, RPCResponse_Result.AsPeerGetInfoResponse, (*RPCResponse_Result).MergePeerGetInfoResponse)
	assertResponseUnion(t, "PeerPutInfo", PeerPutInfoResponse{Name: stringPtr("gear-2")}, (*RPCResponse_Result).FromPeerPutInfoResponse, RPCResponse_Result.AsPeerPutInfoResponse, (*RPCResponse_Result).MergePeerPutInfoResponse)
	assertResponseUnion(t, "PeerGetRuntime", PeerGetRuntimeResponse{Online: true, LastSeenAt: now}, (*RPCResponse_Result).FromPeerGetRuntimeResponse, RPCResponse_Result.AsPeerGetRuntimeResponse, (*RPCResponse_Result).MergePeerGetRuntimeResponse)
	assertResponseUnion(t, "DeviceGetInfo", DeviceGetInfoResponse{Name: stringPtr("gear-1")}, (*RPCResponse_Result).FromDeviceGetInfoResponse, RPCResponse_Result.AsDeviceGetInfoResponse, (*RPCResponse_Result).MergeDeviceGetInfoResponse)
	assertResponseUnion(t, "DeviceGetIdentifiers", DeviceGetIdentifiersResponse{Sn: stringPtr("sn-1")}, (*RPCResponse_Result).FromDeviceGetIdentifiersResponse, RPCResponse_Result.AsDeviceGetIdentifiersResponse, (*RPCResponse_Result).MergeDeviceGetIdentifiersResponse)
	assertResponseUnion(t, "ServerGetInfo", ServerGetInfoResponse{PublicKey: "server", ServerTime: 123}, (*RPCResponse_Result).FromServerGetInfoResponse, RPCResponse_Result.AsServerGetInfoResponse, (*RPCResponse_Result).MergeServerGetInfoResponse)
}

func TestRPCMethodValid(t *testing.T) {
	for _, method := range []RPCMethod{
		RPCMethodPeerPing,
		RPCMethodDeviceInfoGet,
		RPCMethodDeviceIdentifiersGet,
		RPCMethodPeerInfoGet,
		RPCMethodPeerInfoPut,
		RPCMethodPeerRuntimeGet,
		RPCMethodServerInfoGet,
	} {
		if !method.Valid() {
			t.Fatalf("%s should be valid", method)
		}
	}
	if RPCMethod("peer.unknown").Valid() {
		t.Fatal("unknown RPC method should be invalid")
	}
	if !RPCVersionV1.Valid() {
		t.Fatal("RPC version 1 should be valid")
	}
	if RPCVersion(2).Valid() {
		t.Fatal("unknown RPC version should be invalid")
	}
	for _, code := range []RPCErrorCode{RPCErrorCodeInvalidRequest, RPCErrorCodeMethodNotFound, RPCErrorCodeInvalidParams, RPCErrorCodeInternalError, RPCErrorCodeBadRequest, RPCErrorCodeNotFound, RPCErrorCodeConflict} {
		if !code.Valid() {
			t.Fatalf("%d should be valid", code)
		}
	}
	if RPCErrorCode(418).Valid() {
		t.Fatal("unknown RPC error code should be invalid")
	}
}

func assertRequestUnion[T any](
	t *testing.T,
	name string,
	value T,
	from func(*RPCRequest_Params, T) error,
	as func(RPCRequest_Params) (T, error),
	merge func(*RPCRequest_Params, T) error,
) {
	t.Helper()
	var params RPCRequest_Params
	if err := from(&params, value); err != nil {
		t.Fatalf("From%sRequest() error = %v", name, err)
	}
	if _, err := as(params); err != nil {
		t.Fatalf("As%sRequest() error = %v", name, err)
	}
	if err := merge(&params, value); err != nil {
		t.Fatalf("Merge%sRequest() error = %v", name, err)
	}
}

func assertResponseUnion[T any](
	t *testing.T,
	name string,
	value T,
	from func(*RPCResponse_Result, T) error,
	as func(RPCResponse_Result) (T, error),
	merge func(*RPCResponse_Result, T) error,
) {
	t.Helper()
	var result RPCResponse_Result
	if err := from(&result, value); err != nil {
		t.Fatalf("From%sResponse() error = %v", name, err)
	}
	if _, err := as(result); err != nil {
		t.Fatalf("As%sResponse() error = %v", name, err)
	}
	if err := merge(&result, value); err != nil {
		t.Fatalf("Merge%sResponse() error = %v", name, err)
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestWriteFramePropagatesHeaderWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	if err := WriteFrame(errorWriter{err: writeErr}, Frame{Type: FrameTypeJSON, Payload: []byte("payload")}); !errors.Is(err, writeErr) {
		t.Fatalf("WriteFrame() err = %v, want %v", err, writeErr)
	}
}

func TestReadRequestAndResponseRejectInvalidJSON(t *testing.T) {
	var reqBuf bytes.Buffer
	if err := WriteFrame(&reqBuf, Frame{Type: FrameTypeJSON, Payload: []byte("{")}); err != nil {
		t.Fatalf("WriteFrame(request) error = %v", err)
	}
	if _, err := ReadRequest(&reqBuf); err == nil {
		t.Fatal("ReadRequest() should fail for invalid JSON")
	}

	var respBuf bytes.Buffer
	if err := WriteFrame(&respBuf, Frame{Type: FrameTypeJSON, Payload: []byte("{")}); err != nil {
		t.Fatalf("WriteFrame(response) error = %v", err)
	}
	if _, err := ReadResponse(&respBuf); err == nil {
		t.Fatal("ReadResponse() should fail for invalid JSON")
	}
}

func TestReadRequestAndResponseRejectNonJSONFrames(t *testing.T) {
	var reqBuf bytes.Buffer
	if err := WriteFrame(&reqBuf, Frame{Type: FrameTypeBinary, Payload: []byte("{}")}); err != nil {
		t.Fatalf("WriteFrame(request) error = %v", err)
	}
	if _, err := ReadRequest(&reqBuf); err == nil || err.Error() != "rpc: unmarshal request: rpc: expected JSON frame, got type 2" {
		t.Fatalf("ReadRequest() err = %v", err)
	}

	var respBuf bytes.Buffer
	if err := WriteFrame(&respBuf, Frame{Type: FrameTypeText, Payload: []byte("{}")}); err != nil {
		t.Fatalf("WriteFrame(response) error = %v", err)
	}
	if _, err := ReadResponse(&respBuf); err == nil || err.Error() != "rpc: unmarshal response: rpc: expected JSON frame, got type 3" {
		t.Fatalf("ReadResponse() err = %v", err)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

func TestWriteFrameRejectsOversizedFrame(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), MaxFrameSize+1)
	var buf bytes.Buffer
	err := WriteFrame(&buf, Frame{Type: FrameTypeBinary, Payload: payload})
	if err == nil || err.Error() != "rpc: frame too large: 65536" {
		t.Fatalf("WriteFrame() err = %v", err)
	}
}

func TestReadFrameRejectsUnknownType(t *testing.T) {
	var buf bytes.Buffer
	var hdr [4]byte
	binary.LittleEndian.PutUint16(hdr[0:2], 0)
	binary.LittleEndian.PutUint16(hdr[2:4], 99)
	if _, err := buf.Write(hdr[:]); err != nil {
		t.Fatalf("Write(header) error = %v", err)
	}

	_, err := ReadFrame(&buf)
	if err == nil || err.Error() != "rpc: unknown frame type: 99" {
		t.Fatalf("ReadFrame() err = %v", err)
	}
}

func TestReadFrameRejectsTruncatedPayload(t *testing.T) {
	var buf bytes.Buffer
	var hdr [4]byte
	binary.LittleEndian.PutUint16(hdr[0:2], 4)
	binary.LittleEndian.PutUint16(hdr[2:4], uint16(FrameTypeText))
	if _, err := buf.Write(hdr[:]); err != nil {
		t.Fatalf("Write(header) error = %v", err)
	}
	if _, err := buf.Write([]byte("xy")); err != nil {
		t.Fatalf("Write(payload) error = %v", err)
	}

	_, err := ReadFrame(&buf)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ReadFrame() err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestReadWriteFrames(t *testing.T) {
	var buf bytes.Buffer
	err := WriteFrames(&buf, func(yield func(Frame, error) bool) {
		yield(Frame{Type: FrameTypeJSON, Payload: []byte("{}")}, nil)
		yield(Frame{Type: FrameTypeText, Payload: []byte("hello")}, nil)
		yield(Frame{Type: FrameTypeBinary, Payload: []byte{1, 2, 3}}, nil)
	})
	if err != nil {
		t.Fatalf("WriteFrames() error = %v", err)
	}
	if err := WriteEOS(&buf); err != nil {
		t.Fatalf("WriteEOS() error = %v", err)
	}

	var got []Frame
	for frame, err := range ReadFrames(&buf) {
		if err != nil {
			t.Fatalf("ReadFrames() error = %v", err)
		}
		got = append(got, frame)
	}
	if len(got) != 3 {
		t.Fatalf("ReadFrames() got %d frames", len(got))
	}
	if got[0].Type != FrameTypeJSON || string(got[0].Payload) != "{}" {
		t.Fatalf("frame[0] = %+v", got[0])
	}
	if got[1].Type != FrameTypeText || string(got[1].Payload) != "hello" {
		t.Fatalf("frame[1] = %+v", got[1])
	}
	if got[2].Type != FrameTypeBinary || !bytes.Equal(got[2].Payload, []byte{1, 2, 3}) {
		t.Fatalf("frame[2] = %+v", got[2])
	}
}

func TestWriteFramesPropagatesSequenceError(t *testing.T) {
	seqErr := errors.New("sequence failed")
	var buf bytes.Buffer
	err := WriteFrames(&buf, func(yield func(Frame, error) bool) {
		yield(Frame{}, seqErr)
	})
	if !errors.Is(err, seqErr) {
		t.Fatalf("WriteFrames() err = %v, want %v", err, seqErr)
	}
}

func TestReadWriteResponses(t *testing.T) {
	responses := []string{"one", "two"}
	var buf bytes.Buffer
	err := WriteResponses(&buf, func(yield func(*RPCResponse, error) bool) {
		for _, id := range responses {
			if !yield(&RPCResponse{V: RPCVersionV1, Id: id}, nil) {
				return
			}
		}
	})
	if err != nil {
		t.Fatalf("WriteResponses() error = %v", err)
	}

	var got []string
	for resp, err := range ReadResponses(&buf) {
		if err != nil {
			t.Fatalf("ReadResponses() error = %v", err)
		}
		got = append(got, resp.Id)
	}
	if len(got) != len(responses) || got[0] != "one" || got[1] != "two" {
		t.Fatalf("ReadResponses() ids = %v", got)
	}
}

func TestWriteResponsesPropagatesSequenceError(t *testing.T) {
	seqErr := errors.New("response failed")
	var buf bytes.Buffer
	err := WriteResponses(&buf, func(yield func(*RPCResponse, error) bool) {
		yield(nil, seqErr)
	})
	if !errors.Is(err, seqErr) {
		t.Fatalf("WriteResponses() err = %v, want %v", err, seqErr)
	}
}

func TestReadResponsesRejectsNonJSONFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, Frame{Type: FrameTypeBinary, Payload: []byte("{}")}); err != nil {
		t.Fatalf("WriteFrame() error = %v", err)
	}
	for _, err := range ReadResponses(&buf) {
		if err == nil || err.Error() != "rpc: unmarshal response: rpc: expected JSON frame, got type 2" {
			t.Fatalf("ReadResponses() err = %v", err)
		}
		return
	}
	t.Fatal("ReadResponses() did not yield error")
}

func TestReadFramesReturnsEOFBeforeEOS(t *testing.T) {
	var buf bytes.Buffer
	for _, err := range ReadFrames(&buf) {
		if !errors.Is(err, io.EOF) {
			t.Fatalf("ReadFrames() err = %v, want %v", err, io.EOF)
		}
		return
	}
	t.Fatal("ReadFrames() did not yield EOF before EOS")
}

func TestReadFramesStopsOnEOS(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteEOS(&buf); err != nil {
		t.Fatalf("WriteEOS() error = %v", err)
	}
	for range ReadFrames(&buf) {
		t.Fatal("ReadFrames() should not yield EOS")
	}
}

func TestReadWriteEOS(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteEOS(&buf); err != nil {
		t.Fatalf("WriteEOS() error = %v", err)
	}
	if err := ReadEOS(&buf); err != nil {
		t.Fatalf("ReadEOS() error = %v", err)
	}
}

func TestEOSFrameMustBeEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, Frame{Type: FrameTypeEOS, Payload: []byte("x")}); err == nil || err.Error() != "rpc: EOS frame must be empty" {
		t.Fatalf("WriteFrame(EOS payload) err = %v", err)
	}

	var hdr [4]byte
	binary.LittleEndian.PutUint16(hdr[0:2], 1)
	binary.LittleEndian.PutUint16(hdr[2:4], uint16(FrameTypeEOS))
	if _, err := buf.Write(hdr[:]); err != nil {
		t.Fatalf("Write(header) error = %v", err)
	}
	_, err := ReadFrame(&buf)
	if err == nil || err.Error() != "rpc: EOS frame must be empty" {
		t.Fatalf("ReadFrame(EOS payload) err = %v", err)
	}
}

func TestWriteFrameRejectsUnknownType(t *testing.T) {
	var buf bytes.Buffer
	err := WriteFrame(&buf, Frame{Type: FrameType(99), Payload: []byte("payload")})
	if err == nil || err.Error() != "rpc: unknown frame type: 99" {
		t.Fatalf("WriteFrame() err = %v", err)
	}
}

func TestWriteFramePropagatesShortWrite(t *testing.T) {
	err := WriteFrame(shortWriter{}, Frame{Type: FrameTypeJSON, Payload: []byte("payload")})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteFrame() err = %v, want %v", err, io.ErrShortWrite)
	}
}

type shortWriter struct{}

func (shortWriter) Write(_ []byte) (int, error) {
	return 0, nil
}

func TestErrorImplementsErrorAndBuildsRPCResponse(t *testing.T) {
	rpcErr := Error{RequestID: "req-1", Code: RPCErrorCodeInvalidParams, Message: "missing params"}
	var err error = rpcErr
	if err.Error() != "missing params" {
		t.Fatalf("Error() = %q", err.Error())
	}

	errResp := rpcErr.RPCResponse()
	if errResp.V != RPCVersionV1 || errResp.Id != "req-1" || errResp.Error == nil {
		t.Fatalf("RPCResponse() = %+v", errResp)
	}
	if errResp.Error.Code != RPCErrorCodeInvalidParams || errResp.Error.Message != "missing params" {
		t.Fatalf("RPCResponse().Error = %+v", errResp.Error)
	}
}

func TestErrorUsesFallbackMessage(t *testing.T) {
	rpcErr := Error{Code: RPCErrorCode(-1)}
	if rpcErr.Error() != "rpc error -1" {
		t.Fatalf("Error() = %q", rpcErr.Error())
	}

	errResp := rpcErr.RPCResponse()
	if errResp.Error == nil || errResp.Error.Message != "rpc error -1" {
		t.Fatalf("RPCResponse().Error = %+v", errResp.Error)
	}
}
