package gizcli

import (
	"net"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/giznoise"
)

type testClientListener struct{}

func (testClientListener) Accept() (giznet.Conn, error) { return nil, giznet.ErrClosed }
func (testClientListener) Close() error                 { return nil }

func testNoiseDialTransport() DialTransportFunc {
	return func(key *giznet.KeyPair, serverPK giznet.PublicKey, serverAddr string, securityPolicy giznet.SecurityPolicy) (giznet.Listener, giznet.Conn, error) {
		listener, err := (&giznoise.ListenConfig{
			Addr:           ":0",
			SecurityPolicy: securityPolicy,
		}).Listen(key)
		if err != nil {
			return nil, nil, err
		}
		udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
		if err != nil {
			_ = listener.Close()
			return nil, nil, err
		}
		conn, err := listener.Dial(serverPK, udpAddr)
		if err != nil {
			_ = listener.Close()
			return nil, nil, err
		}
		return listener, conn, nil
	}
}
