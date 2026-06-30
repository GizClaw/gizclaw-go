package giznet

import (
	"net"
	"testing"
)

type fakeConn struct{}

func (fakeConn) Dial(uint64) (net.Conn, error) { return nil, nil }
func (fakeConn) ListenService(uint64) ServiceListener {
	return fakeServiceListener{}
}
func (fakeConn) CloseService(uint64) error            { return nil }
func (fakeConn) Read([]byte) (byte, int, error)       { return 0, 0, nil }
func (fakeConn) Write(byte, []byte) (int, error)      { return 0, nil }
func (fakeConn) PublicKey() PublicKey                 { return PublicKey{} }
func (fakeConn) PeerInfo() *PeerInfo                  { return nil }
func (fakeConn) Close() error                         { return nil }
func (fakeServiceListener) Accept() (net.Conn, error) { return nil, nil }
func (fakeServiceListener) Close() error              { return nil }
func (fakeServiceListener) Addr() net.Addr            { return nil }
func (fakeListener) Accept() (Conn, error)            { return fakeConn{}, nil }
func (fakeListener) Close() error                     { return nil }

type fakeServiceListener struct{}
type fakeListener struct{}

func TestInterfaceContracts(t *testing.T) {
	var _ Conn = fakeConn{}
	var _ ServiceListener = fakeServiceListener{}
	var _ Listener = fakeListener{}
}
