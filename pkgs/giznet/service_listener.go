package giznet

import "net"

// ServiceListener accepts streams for a single giznet service.
type ServiceListener interface {
	Accept() (net.Conn, error)
	Close() error
	Addr() net.Addr
}
