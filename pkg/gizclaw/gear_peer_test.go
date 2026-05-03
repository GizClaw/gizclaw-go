package gizclaw

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpc"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
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

func TestGearPeerCloseClosesConn(t *testing.T) {
	serverKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(server) error = %v", err)
	}
	clientKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(client) error = %v", err)
	}
	serverListener, err := giznet.Listen(serverKey, giznet.WithBindAddr("127.0.0.1:0"), giznet.WithAllowUnknown(true))
	if err != nil {
		t.Fatalf("Listen(server) error = %v", err)
	}
	defer serverListener.Close()
	go drainUDP(serverListener.UDP())
	clientListener, err := giznet.Listen(clientKey, giznet.WithBindAddr("127.0.0.1:0"), giznet.WithAllowUnknown(true))
	if err != nil {
		t.Fatalf("Listen(client) error = %v", err)
	}
	defer clientListener.Close()
	go drainUDP(clientListener.UDP())

	acceptCh := make(chan *giznet.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := serverListener.Accept()
		if err != nil {
			errCh <- err
			return
		}
		acceptCh <- conn
	}()

	clientConn, err := clientListener.Dial(serverKey.Public, serverListener.HostInfo().Addr)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}
	defer clientConn.Close()

	var serverConn *giznet.Conn
	select {
	case serverConn = <-acceptCh:
	case err := <-errCh:
		t.Fatalf("Accept error = %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Accept timeout")
	}

	peer := &GearPeer{Conn: serverConn}
	if err := peer.close(); err != nil {
		t.Fatalf("GearPeer.close() error = %v", err)
	}
	if err := serverConn.Close(); !errors.Is(err, giznet.ErrConnClosed) {
		t.Fatalf("server Conn.Close() after GearPeer.close err=%v, want %v", err, giznet.ErrConnClosed)
	}
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
