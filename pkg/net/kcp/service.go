package kcp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrServiceMuxClosed  = errors.New("kcp: service mux closed")
	ErrServiceNotFound   = errors.New("kcp: service not found")
	ErrServiceRejected   = errors.New("kcp: service rejected")
	ErrAcceptQueueClosed = errors.New("kcp: accept queue closed")
)

// ServiceMuxConfig holds configuration for ServiceMux.
type ServiceMuxConfig struct {
	// IsClient is kept for API compatibility. Direct KCP stream mode
	// does not require separate client/server session roles.
	IsClient bool

	// Output is called to send KCP packets over the wire.
	Output func(service uint64, data []byte) error

	// OnOutputError is called when Output returns an error.
	OnOutputError func(service uint64, err error)

	// OnNewService is called when a service is first created.
	// Return true to accept, false to reject.
	OnNewService func(service uint64) bool
}

type serviceEntry struct {
	conn      *KCPConn
	announced atomic.Bool
	readyOnce sync.Once
	readyCh   chan struct{}
}

// ServiceMux manages per-service KCP streams for a peer.
//
// Compared with yamux mode, each service now maps to exactly one direct KCPConn.
// OpenStream returns that KCPConn directly. AcceptStream is triggered when
// inbound packets are first seen on a service.
type ServiceMux struct {
	config ServiceMuxConfig

	services   map[uint64]*serviceEntry
	servicesMu sync.RWMutex

	acceptCh chan acceptResult
	closeCh  chan struct{}

	closed    atomic.Bool
	closeOnce sync.Once

	outputErrors atomic.Uint64
}

type acceptResult struct {
	conn    net.Conn
	service uint64
}

// directStream is a logical stream wrapper on top of a shared per-service KCPConn.
// Close is a no-op to preserve request-scoped close semantics without tearing down
// the underlying long-lived KCP channel.
type directStream struct {
	conn net.Conn
}

func (s *directStream) Read(b []byte) (int, error)         { return s.conn.Read(b) }
func (s *directStream) Write(b []byte) (int, error)        { return s.conn.Write(b) }
func (s *directStream) Close() error                       { return nil }
func (s *directStream) LocalAddr() net.Addr                { return s.conn.LocalAddr() }
func (s *directStream) RemoteAddr() net.Addr               { return s.conn.RemoteAddr() }
func (s *directStream) SetDeadline(t time.Time) error      { return s.conn.SetDeadline(t) }
func (s *directStream) SetReadDeadline(t time.Time) error  { return s.conn.SetReadDeadline(t) }
func (s *directStream) SetWriteDeadline(t time.Time) error { return s.conn.SetWriteDeadline(t) }

func wrapDirectStream(conn net.Conn) net.Conn {
	if conn == nil {
		return nil
	}
	return &directStream{conn: conn}
}

// NewServiceMux creates a new ServiceMux.
func NewServiceMux(cfg ServiceMuxConfig) *ServiceMux {
	return &ServiceMux{
		config:   cfg,
		services: make(map[uint64]*serviceEntry),
		acceptCh: make(chan acceptResult, 4096),
		closeCh:  make(chan struct{}),
	}
}

// Input routes an incoming KCP packet to the correct service KCPConn.
func (m *ServiceMux) Input(service uint64, data []byte) error {
	if m.closed.Load() {
		return ErrServiceMuxClosed
	}

	entry, err := m.getOrCreateService(service)
	if err != nil {
		return err
	}

	if err := entry.conn.Input(data); err != nil {
		if !errors.Is(err, ErrConnClosed) {
			return err
		}
		entry, err = m.recreateService(service, entry)
		if err != nil {
			return err
		}
		if err := entry.conn.Input(data); err != nil {
			return err
		}
	}

	m.announceAccept(service, entry)
	return nil
}

// OpenStream returns the direct KCP stream for a service.
func (m *ServiceMux) OpenStream(service uint64) (net.Conn, error) {
	if m.closed.Load() {
		return nil, ErrServiceMuxClosed
	}

	entry, err := m.getOrCreateService(service)
	if err != nil {
		return nil, err
	}

	if entry.conn.IsClosed() {
		entry, err = m.recreateService(service, entry)
		if err != nil {
			return nil, err
		}
	}

	return wrapDirectStream(entry.conn), nil
}

// AcceptStream accepts the next inbound service stream.
// A service is announced when inbound packets are first observed.
func (m *ServiceMux) AcceptStream() (net.Conn, uint64, error) {
	if m.closed.Load() {
		return nil, 0, ErrServiceMuxClosed
	}

	// Fast path: if any service has been announced, return it directly.
	m.servicesMu.RLock()
	for svc, entry := range m.services {
		if entry != nil && entry.announced.Load() {
			conn := entry.conn
			m.servicesMu.RUnlock()
			return wrapDirectStream(conn), svc, nil
		}
	}
	m.servicesMu.RUnlock()

	result, ok := <-m.acceptCh
	if !ok {
		return nil, 0, ErrAcceptQueueClosed
	}
	return wrapDirectStream(result.conn), result.service, nil
}

// AcceptStreamOn waits for inbound activity on a specific service and
// returns the corresponding direct KCP stream.
func (m *ServiceMux) AcceptStreamOn(service uint64) (net.Conn, error) {
	if m.closed.Load() {
		return nil, ErrServiceMuxClosed
	}

	entry, err := m.getOrCreateService(service)
	if err != nil {
		return nil, err
	}

	if entry.announced.Load() {
		return entry.conn, nil
	}

	select {
	case <-entry.readyCh:
		return wrapDirectStream(entry.conn), nil
	case <-m.closeCh:
		return nil, ErrServiceMuxClosed
	}
}

// Close closes all service streams.
func (m *ServiceMux) Close() error {
	m.closeOnce.Do(func() {
		m.closed.Store(true)
		close(m.closeCh)

		m.servicesMu.Lock()
		entries := make([]*serviceEntry, 0, len(m.services))
		for _, e := range m.services {
			entries = append(entries, e)
		}
		m.services = make(map[uint64]*serviceEntry)
		m.servicesMu.Unlock()

		for _, e := range entries {
			_ = e.conn.Close()
		}

		close(m.acceptCh)
	})
	return nil
}

// NumServices returns the number of active services.
func (m *ServiceMux) NumServices() int {
	m.servicesMu.RLock()
	defer m.servicesMu.RUnlock()
	return len(m.services)
}

// NumStreams returns the number of direct KCP streams.
// In direct mode, one service corresponds to one stream.
func (m *ServiceMux) NumStreams() int {
	return m.NumServices()
}

// OutputErrorCount returns the number of output callback failures observed.
func (m *ServiceMux) OutputErrorCount() uint64 {
	return m.outputErrors.Load()
}

func (m *ServiceMux) reportOutputError(service uint64, err error) {
	if err == nil {
		return
	}
	m.outputErrors.Add(1)
	if m.config.OnOutputError != nil {
		m.config.OnOutputError(service, err)
	}
}

func (m *ServiceMux) announceAccept(service uint64, entry *serviceEntry) {
	if !entry.announced.CompareAndSwap(false, true) {
		return
	}

	entry.readyOnce.Do(func() {
		close(entry.readyCh)
	})

	select {
	case m.acceptCh <- acceptResult{conn: entry.conn, service: service}:
	case <-m.closeCh:
	}
}

func (m *ServiceMux) getOrCreateService(service uint64) (*serviceEntry, error) {
	m.servicesMu.RLock()
	entry, ok := m.services[service]
	m.servicesMu.RUnlock()
	if ok && !entry.conn.IsClosed() {
		return entry, nil
	}

	m.servicesMu.Lock()
	defer m.servicesMu.Unlock()

	if entry, ok := m.services[service]; ok {
		if !entry.conn.IsClosed() {
			return entry, nil
		}
		_ = entry.conn.Close()
		delete(m.services, service)
	}

	if m.closed.Load() {
		return nil, ErrServiceMuxClosed
	}

	if m.config.OnNewService != nil && !m.config.OnNewService(service) {
		return nil, ErrServiceRejected
	}

	entry = m.createServiceLocked(service)
	m.services[service] = entry
	return entry, nil
}

func (m *ServiceMux) recreateService(service uint64, stale *serviceEntry) (*serviceEntry, error) {
	m.servicesMu.Lock()
	defer m.servicesMu.Unlock()

	if m.closed.Load() {
		return nil, ErrServiceMuxClosed
	}

	current, ok := m.services[service]
	if ok && current != stale && !current.conn.IsClosed() {
		return current, nil
	}

	if ok {
		_ = current.conn.Close()
	}

	entry := m.createServiceLocked(service)
	m.services[service] = entry
	return entry, nil
}

func (m *ServiceMux) createServiceLocked(service uint64) *serviceEntry {
	conn := NewKCPConn(uint32(service), func(data []byte) {
		if m.config.Output == nil {
			return
		}
		if err := m.config.Output(service, data); err != nil {
			m.reportOutputError(service, err)
		}
	})

	return &serviceEntry{
		conn:    conn,
		readyCh: make(chan struct{}),
	}
}
