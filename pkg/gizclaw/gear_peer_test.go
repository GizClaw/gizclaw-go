package gizclaw

import (
	"context"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpc"
)

func TestGearPeerHelpersAndDispatch(t *testing.T) {
	t.Run("audio mixer lifecycle", func(t *testing.T) {
		var nilPeer *GearPeer
		if _, err := nilPeer.audioMixer(); err != ErrNilGearPeer {
			t.Fatalf("audioMixer(nil) err = %v, want %v", err, ErrNilGearPeer)
		}

		peer := &GearPeer{}
		if _, err := peer.audioMixer(); err != ErrNilGearPeerMixer {
			t.Fatalf("audioMixer() err = %v, want %v", err, ErrNilGearPeerMixer)
		}

		peer.init()
		if _, err := peer.audioMixer(); err != nil {
			t.Fatalf("audioMixer() after init error = %v", err)
		}

		track, ctrl, err := peer.CreateAudioTrack()
		if err != nil {
			t.Fatalf("CreateAudioTrack() error = %v", err)
		}
		if track == nil || ctrl == nil {
			t.Fatalf("CreateAudioTrack() = (%v, %v)", track, ctrl)
		}
		if err := peer.close(); err != nil {
			t.Fatalf("close() error = %v", err)
		}
		if !peer.isClosed() {
			t.Fatal("peer should be closed")
		}
	})

	t.Run("dispatch missing params", func(t *testing.T) {
		resp, err := (&GearPeer{}).dispatchRPC(context.Background(), &rpc.RPCRequest{
			Id:     "missing",
			Method: rpc.MethodPing,
		})
		if err != nil {
			t.Fatalf("dispatchRPC() error = %v", err)
		}
		if resp == nil || resp.Error == nil || resp.Error.Code != -32602 {
			t.Fatalf("dispatchRPC() response = %+v", resp)
		}
	})

	t.Run("dispatch ping and unknown method", func(t *testing.T) {
		peer := &GearPeer{}
		resp, err := peer.dispatchRPC(context.Background(), &rpc.RPCRequest{
			Id:     "ping",
			Method: rpc.MethodPing,
			Params: &rpc.PingRequest{},
		})
		if err != nil {
			t.Fatalf("dispatchRPC(ping) error = %v", err)
		}
		if resp == nil || resp.Result == nil || resp.Result.ServerTime <= 0 {
			t.Fatalf("dispatchRPC(ping) response = %+v", resp)
		}

		resp, err = peer.dispatchRPC(context.Background(), &rpc.RPCRequest{
			Id:     "unknown",
			Method: "rpc.unknown",
		})
		if err != nil {
			t.Fatalf("dispatchRPC(unknown) error = %v", err)
		}
		if resp == nil || resp.Error == nil || !strings.Contains(resp.Error.Message, "unknown method") {
			t.Fatalf("dispatchRPC(unknown) response = %+v", resp)
		}
	})
}

func TestGearPeerPCMChunkToInt16(t *testing.T) {
	chunk := &pcm.DataChunk{Data: []byte{0x34, 0x12, 0x78, 0x56}}
	got := gearPeerPCMChunkToInt16(chunk)
	if len(got) != 2 {
		t.Fatalf("len(gearPeerPCMChunkToInt16()) = %d", len(got))
	}
	if got[0] != 0x1234 || got[1] != 0x5678 {
		t.Fatalf("gearPeerPCMChunkToInt16() = %#v", got)
	}
	if out := gearPeerPCMChunkToInt16(nil); out != nil {
		t.Fatalf("gearPeerPCMChunkToInt16(nil) = %#v", out)
	}
}
