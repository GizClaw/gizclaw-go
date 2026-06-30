package giznet

import (
	"net"
)

// Conn is the transport-independent peer connection surface used by gizclaw.
type Conn interface {
	Dial(service uint64) (net.Conn, error)
	ListenService(service uint64) ServiceListener
	CloseService(service uint64) error

	Read([]byte) (protocol byte, n int, err error)
	Write(protocol byte, payload []byte) (int, error)

	PublicKey() PublicKey
	PeerInfo() *PeerInfo
	Close() error
}
