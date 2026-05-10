package gizclaw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"

	"github.com/GizClaw/gizclaw-go/pkg/audio/stampedopus"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpcapi"
)

const (
	WebRTCDataChannelRPCLabel = "rpc"

	webRTCAudioTrackID    = "gizclaw-audio"
	webRTCAudioStreamID   = "gizclaw"
	webRTCOpusClockRate   = 48000
	webRTCOpusPayloadType = 111
	webRTCRPCTimeout      = 30 * time.Second
)

// ClientWebRTCRegistration is the live bridge between one Pion PeerConnection
// and the connected GizClaw peer transport.
type ClientWebRTCRegistration struct {
	client *Client
	pc     *webrtc.PeerConnection

	ctx    context.Context
	cancel context.CancelFunc

	audioTrack  *webrtc.TrackLocalStaticRTP
	audioSender *webrtc.RTPSender
}

// RegisterTo wires this client into a Pion PeerConnection.
//
// The browser-facing contract is intentionally transport-shaped rather than
// signaling-shaped: cmd/play can use any local signaling mechanism, then call
// RegisterTo before applying the offer/answer.
func (c *Client) RegisterTo(pc *webrtc.PeerConnection) (*ClientWebRTCRegistration, error) {
	if c == nil {
		return nil, fmt.Errorf("gizclaw: nil client")
	}
	if pc == nil {
		return nil, fmt.Errorf("gizclaw: nil peer connection")
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: webRTCOpusClockRate,
			Channels:  2,
		},
		webRTCAudioTrackID,
		webRTCAudioStreamID,
	)
	if err != nil {
		return nil, fmt.Errorf("gizclaw: create webrtc audio track: %w", err)
	}

	audioSender, err := pc.AddTrack(audioTrack)
	if err != nil {
		return nil, fmt.Errorf("gizclaw: add webrtc audio track: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &ClientWebRTCRegistration{
		client:      c,
		pc:          pc,
		ctx:         ctx,
		cancel:      cancel,
		audioTrack:  audioTrack,
		audioSender: audioSender,
	}

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		r.registerDataChannel(dc)
	})
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		r.registerRemoteTrack(track)
	})

	go r.forwardPeerStampedOpusToWebRTCAudio()
	go drainWebRTCRTCP(audioSender)

	return r, nil
}

type stampedOpusFrame struct {
	timestamp uint64
	frame     []byte
}

type stampedOpusJitterBuffer struct {
	frames []stampedOpusFrame
	depth  int
}

func newStampedOpusJitterBuffer(depth int) *stampedOpusJitterBuffer {
	if depth < 1 {
		depth = 1
	}
	return &stampedOpusJitterBuffer{depth: depth}
}

func (b *stampedOpusJitterBuffer) Push(timestamp uint64, frame []byte) []stampedOpusFrame {
	if b == nil {
		return []stampedOpusFrame{{timestamp: timestamp, frame: append([]byte(nil), frame...)}}
	}
	b.frames = append(b.frames, stampedOpusFrame{timestamp: timestamp, frame: append([]byte(nil), frame...)})
	sort.SliceStable(b.frames, func(i, j int) bool {
		return b.frames[i].timestamp < b.frames[j].timestamp
	})
	if len(b.frames) <= b.depth {
		return nil
	}
	out := b.frames[:1]
	b.frames = b.frames[1:]
	return out
}

// AudioTrack returns the local WebRTC audio track that receives server-side
// stamped opus packets.
func (r *ClientWebRTCRegistration) AudioTrack() *webrtc.TrackLocalStaticRTP {
	if r == nil {
		return nil
	}
	return r.audioTrack
}

// Close stops registration-owned forwarding loops. It does not close the
// PeerConnection or the GizClaw Client.
func (r *ClientWebRTCRegistration) Close() error {
	if r == nil {
		return nil
	}
	r.cancel()
	if r.pc != nil && r.audioSender != nil {
		return r.pc.RemoveTrack(r.audioSender)
	}
	return nil
}

func (r *ClientWebRTCRegistration) registerDataChannel(dc *webrtc.DataChannel) {
	if r == nil || dc == nil || !isWebRTCRPCDataChannel(dc.Label()) {
		return
	}
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		go func() {
			r.handleRPCDataChannelMessage(dc, msg)
		}()
	})
}

func isWebRTCRPCDataChannel(label string) bool {
	return label == WebRTCDataChannelRPCLabel || strings.HasPrefix(label, WebRTCDataChannelRPCLabel+":")
}

func (r *ClientWebRTCRegistration) handleRPCDataChannelMessage(dc *webrtc.DataChannel, msg webrtc.DataChannelMessage) {
	if len(msg.Data) > rpcapi.MaxFrameSize {
		r.sendRPCDataChannelResponse(dc, msg.IsString, rpcapi.Error{Code: rpcapi.RPCErrorCodeInvalidRequest, Message: "rpc message too large"}.RPCResponse())
		return
	}

	var req rpcapi.RPCRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		r.sendRPCDataChannelResponse(dc, msg.IsString, rpcapi.Error{Code: rpcapi.RPCErrorCode(-32700), Message: fmt.Sprintf("invalid rpc json: %v", err)}.RPCResponse())
		return
	}

	ctx, cancel := context.WithTimeout(r.ctx, webRTCRPCTimeout)
	defer cancel()

	resp, err := r.client.callRPCRequest(ctx, &req)
	if err != nil {
		resp = rpcapi.Error{RequestID: req.Id, Code: rpcapi.RPCErrorCode(-32000), Message: err.Error()}.RPCResponse()
	}
	r.sendRPCDataChannelResponse(dc, msg.IsString, resp)
}

func (r *ClientWebRTCRegistration) sendRPCDataChannelResponse(dc *webrtc.DataChannel, asString bool, resp *rpcapi.RPCResponse) {
	if dc == nil || resp == nil {
		return
	}
	defer func() {
		if err := dc.Close(); err != nil {
			slog.Debug("gizclaw: close webrtc rpc data channel failed", "error", err)
		}
	}()
	if resp.V == 0 {
		resp.V = rpcapi.RPCVersionV1
	}

	data, err := json.Marshal(resp)
	if err != nil {
		slog.Debug("gizclaw: marshal webrtc rpc response failed", "error", err)
		return
	}
	if asString {
		if err := dc.SendText(string(data)); err != nil {
			slog.Debug("gizclaw: send webrtc rpc text response failed", "error", err)
		}
		return
	}
	if err := dc.Send(data); err != nil {
		slog.Debug("gizclaw: send webrtc rpc binary response failed", "error", err)
	}
}

func (r *ClientWebRTCRegistration) registerRemoteTrack(track *webrtc.TrackRemote) {
	if r == nil || track == nil {
		return
	}

	codec := track.Codec()
	switch {
	case track.Kind() == webrtc.RTPCodecTypeAudio && strings.EqualFold(codec.MimeType, webrtc.MimeTypeOpus):
		go func() {
			if err := r.forwardWebRTCAudioTrackToPeerStampedOpus(track); err != nil && !errors.Is(err, context.Canceled) {
				slog.Debug("gizclaw: forward webrtc opus track failed", "error", err)
			}
		}()
	default:
		go func() {
			drainWebRTCRemoteTrack(r.ctx, track)
		}()
	}
}

func (r *ClientWebRTCRegistration) forwardWebRTCAudioTrackToPeerStampedOpus(track *webrtc.TrackRemote) error {
	if track == nil {
		return nil
	}
	conn := r.client.PeerConn()
	if conn == nil {
		return fmt.Errorf("gizclaw: client is not connected")
	}
	var (
		baseRTPTimestamp uint32
		baseWallMillis   uint64
		haveBase         bool
	)
	for {
		if err := r.ctx.Err(); err != nil {
			return err
		}

		packet, _, err := track.ReadRTP()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if len(packet.Payload) == 0 {
			continue
		}
		if !haveBase {
			baseRTPTimestamp = packet.Timestamp
			baseWallMillis = uint64(time.Now().UnixMilli())
			haveBase = true
		}

		timestamp := baseWallMillis + webRTCRTPMillisDelta(webRTCOpusClockRate, baseRTPTimestamp, packet.Timestamp)
		payload := stampedopus.Pack(timestamp, packet.Payload)
		if _, err := conn.Write(ProtocolStampedOpus, payload); err != nil {
			return err
		}
	}
}

func (r *ClientWebRTCRegistration) forwardPeerStampedOpusToWebRTCAudio() {
	packets, unsubscribe := r.client.subscribePeerPackets(ProtocolStampedOpus, 32)
	defer unsubscribe()

	var sequenceNumber uint16
	jitter := newStampedOpusJitterBuffer(4)
	for {
		select {
		case <-r.ctx.Done():
			return
		case payload := <-packets:
			timestamp, frame, ok := stampedopus.Unpack(payload)
			if !ok {
				continue
			}
			for _, item := range jitter.Push(timestamp, frame) {
				packet := &rtp.Packet{
					Header: rtp.Header{
						Version:        2,
						PayloadType:    webRTCOpusPayloadType,
						SequenceNumber: sequenceNumber,
						Timestamp:      webRTCOpusRTPTimestamp(item.timestamp),
					},
					Payload: item.frame,
				}
				if err := r.audioTrack.WriteRTP(packet); err != nil {
					slog.Debug("gizclaw: write webrtc opus rtp failed", "error", err)
					return
				}
				sequenceNumber++
			}
		}
	}
}

func (c *Client) callRPCRequest(ctx context.Context, req *rpcapi.RPCRequest) (*rpcapi.RPCResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("rpc: nil request")
	}
	if req.Id == "" {
		return nil, fmt.Errorf("rpc: request id required")
	}
	stream, err := c.rpcConn()
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()
	return callRPC(ctx, stream, req)
}

func webRTCRTPMillisDelta(clockRate uint32, baseTimestamp, timestamp uint32) uint64 {
	if clockRate == 0 {
		return 0
	}
	return uint64(timestamp-baseTimestamp) * uint64(time.Second/time.Millisecond) / uint64(clockRate)
}

func webRTCOpusRTPTimestamp(stampedMillis uint64) uint32 {
	return uint32(stampedMillis * uint64(webRTCOpusClockRate) / uint64(time.Second/time.Millisecond))
}

func drainWebRTCRTCP(sender *webrtc.RTPSender) {
	if sender == nil {
		return
	}
	buf := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buf); err != nil {
			return
		}
	}
}

func drainWebRTCRemoteTrack(ctx context.Context, track *webrtc.TrackRemote) {
	if track == nil {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		if _, _, err := track.ReadRTP(); err != nil {
			return
		}
	}
}
