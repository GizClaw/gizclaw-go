package giznet

// Listener accepts transport-independent peer connections.
type Listener interface {
	Accept() (Conn, error)
	Close() error
}
