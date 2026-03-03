package kcp

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
)

var (
	ErrServiceMuxClosed    = errors.New("kcp: service mux closed")
	ErrServiceNotFound     = errors.New("kcp: service not found")
	ErrServiceRejected     = errors.New("kcp: service rejected")
	ErrAcceptQueueClosed   = errors.New("kcp: accept queue closed")
	ErrInvalidServiceFrame = errors.New("kcp: invalid service frame")
)

const (
	serviceFrameData byte = iota
	serviceFrameClose
	serviceFrameCloseAck
)

const (
	serviceCloseReasonLocal byte = iota
	serviceCloseReasonPeer
)

const (
	activeCloseAckTimeout    = 400 * time.Millisecond
	activeCloseRetryInterval = 100 * time.Millisecond
)

// ServiceMuxConfig holds configuration for ServiceMux.
type ServiceMuxConfig struct {
	// IsClient determines yamux role. Client initiates streams with odd IDs.
	IsClient bool

	// Output is called to send KCP packets over the wire.
	// The service ID is included so the caller can prepend it to the packet.
	Output func(service uint64, data []byte) error

	// OnOutputError is called when Output returns an error.
	// Optional; can be used for metrics/monitoring/degradation policy.
	OnOutputError func(service uint64, err error)

	// OnNewService is called when a new service is seen for the first time
	// (from incoming data). Return true to accept, false to reject.
	// If nil, all services are accepted.
	OnNewService func(service uint64) bool

	// YamuxConfig is the yamux session configuration. nil uses defaults.
	YamuxConfig *yamux.Config
}

// serviceEntry holds one service's KCPConn + yamux session.
type serviceEntry struct {
	conn    *KCPConn
	session *yamux.Session
	pipe    *kcpPipe
}

// ServiceMux manages per-service KCP instances and yamux sessions for a peer.
//
// Each service gets its own KCPConn (reliable byte stream with independent
// goroutine) and yamux.Session (stream multiplexing over the KCPConn).
// Different services are completely isolated at the KCP level.
type ServiceMux struct {
	config ServiceMuxConfig

	services   map[uint64]*serviceEntry
	servicesMu sync.RWMutex

	acceptCh chan acceptResult
	closeCh  chan struct{}

	closing   atomic.Bool
	closed    atomic.Bool
	closeOnce sync.Once
	acceptWg  sync.WaitGroup

	outputErrors atomic.Uint64
	closeSeq     atomic.Uint64

	closeAckMu      sync.Mutex
	closeAckWaiters map[closeToken]chan struct{}
}

type acceptResult struct {
	conn    net.Conn
	service uint64
}

type closeToken struct {
	service uint64
	id      uint64
}

// NewServiceMux creates a new ServiceMux.
func NewServiceMux(cfg ServiceMuxConfig) *ServiceMux {
	return &ServiceMux{
		config:          cfg,
		services:        make(map[uint64]*serviceEntry),
		acceptCh:        make(chan acceptResult, 4096),
		closeCh:         make(chan struct{}),
		closeAckWaiters: make(map[closeToken]chan struct{}),
	}
}

// Input routes an incoming KCP packet to the correct service's KCPConn.
// If the service doesn't exist and OnNewService allows it, creates it.
func (m *ServiceMux) Input(service uint64, data []byte) error {
	if len(data) == 0 {
		return ErrInvalidServiceFrame
	}

	frameType := data[0]
	payload := data[1:]

	switch frameType {
	case serviceFrameData:
		if m.closed.Load() {
			return ErrServiceMuxClosed
		}
		return m.handleDataFrame(service, payload)
	case serviceFrameClose:
		return m.handleCloseFrame(service, payload)
	case serviceFrameCloseAck:
		return m.handleCloseAckFrame(service, payload)
	default:
		return ErrInvalidServiceFrame
	}
}

func (m *ServiceMux) handleDataFrame(service uint64, payload []byte) error {
	if m.closing.Load() {
		return ErrServiceMuxClosed
	}

	entry, err := m.getOrCreateService(service)
	if err != nil {
		return err
	}
	return entry.conn.Input(payload)
}

func (m *ServiceMux) handleCloseFrame(service uint64, payload []byte) error {
	if len(payload) < 9 {
		return ErrInvalidServiceFrame
	}

	closeID := binary.BigEndian.Uint64(payload[:8])
	_ = payload[8] // reason 保留给上层可观测性扩展

	m.sendCloseAck(service, closeID)
	m.closeService(service, serviceCloseReasonPeer)
	return nil
}

func (m *ServiceMux) handleCloseAckFrame(service uint64, payload []byte) error {
	if len(payload) < 8 {
		return ErrInvalidServiceFrame
	}

	closeID := binary.BigEndian.Uint64(payload[:8])
	m.notifyCloseAck(service, closeID)
	return nil
}

// OpenStream opens a new yamux stream on the given service.
// If the service doesn't exist yet, creates KCPConn + yamux session.
func (m *ServiceMux) OpenStream(service uint64) (net.Conn, error) {
	if m.closed.Load() || m.closing.Load() {
		return nil, ErrServiceMuxClosed
	}

	entry, err := m.getOrCreateService(service)
	if err != nil {
		return nil, err
	}
	return entry.session.Open()
}

// AcceptStream accepts the next incoming yamux stream from any service.
// Returns the stream and its service ID.
func (m *ServiceMux) AcceptStream() (net.Conn, uint64, error) {
	if m.closed.Load() || m.closing.Load() {
		return nil, 0, ErrServiceMuxClosed
	}

	result, ok := <-m.acceptCh
	if !ok {
		return nil, 0, ErrAcceptQueueClosed
	}
	return result.conn, result.service, nil
}

// AcceptStreamOn accepts the next incoming yamux stream on a specific service.
func (m *ServiceMux) AcceptStreamOn(service uint64) (net.Conn, error) {
	if m.closed.Load() || m.closing.Load() {
		return nil, ErrServiceMuxClosed
	}

	entry, err := m.getOrCreateService(service)
	if err != nil {
		return nil, err
	}
	return entry.session.Accept()
}

// Close closes all services, KCPConns, and yamux sessions.
func (m *ServiceMux) Close() error {
	m.closeOnce.Do(func() {
		m.closed.Store(true)
		m.closing.Store(true)
		close(m.closeCh)

		services := m.detachServices()
		m.activeCloseServices(services)

		for _, entry := range services {
			m.closeServiceEntry(entry, serviceCloseReasonLocal)
		}
		m.clearCloseAckWaiters()

		m.acceptWg.Wait()
		close(m.acceptCh)
	})
	return nil
}

func (m *ServiceMux) detachServices() map[uint64]*serviceEntry {
	m.servicesMu.Lock()
	defer m.servicesMu.Unlock()

	detached := make(map[uint64]*serviceEntry, len(m.services))
	for service, entry := range m.services {
		detached[service] = entry
	}
	m.services = make(map[uint64]*serviceEntry)
	return detached
}

func (m *ServiceMux) closeService(service uint64, reason byte) {
	m.servicesMu.Lock()
	entry, ok := m.services[service]
	if ok {
		delete(m.services, service)
	}
	m.servicesMu.Unlock()

	if ok {
		m.closeServiceEntry(entry, reason)
	}
}

func (m *ServiceMux) closeServiceEntry(entry *serviceEntry, reason byte) {
	if entry == nil {
		return
	}
	closeErr := ErrConnClosedLocal
	if reason == serviceCloseReasonPeer {
		closeErr = ErrConnClosedByPeer
	}
	_ = entry.conn.closeWithReason(closeErr)
	entry.session.Close()
	entry.pipe.Close()
}

func (m *ServiceMux) activeCloseServices(services map[uint64]*serviceEntry) {
	if len(services) == 0 {
		return
	}

	var wg sync.WaitGroup
	for service := range services {
		svc := service
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.sendCloseAndWaitAck(svc)
		}()
	}
	wg.Wait()
}

func (m *ServiceMux) sendCloseAndWaitAck(service uint64) {
	if m.config.Output == nil {
		return
	}

	closeID := m.closeSeq.Add(1)
	waiter := m.registerCloseAck(service, closeID)
	defer m.unregisterCloseAck(service, closeID)

	m.sendCloseFrame(service, closeID, serviceCloseReasonLocal)

	deadline := time.NewTimer(activeCloseAckTimeout)
	retry := time.NewTicker(activeCloseRetryInterval)
	defer deadline.Stop()
	defer retry.Stop()

	for {
		select {
		case <-waiter:
			return
		case <-retry.C:
			m.sendCloseFrame(service, closeID, serviceCloseReasonLocal)
		case <-deadline.C:
			return
		}
	}
}

func (m *ServiceMux) sendCloseFrame(service uint64, closeID uint64, reason byte) {
	frame := make([]byte, 1+8+1)
	frame[0] = serviceFrameClose
	binary.BigEndian.PutUint64(frame[1:9], closeID)
	frame[9] = reason
	if err := m.config.Output(service, frame); err != nil {
		m.reportOutputError(service, err)
	}
}

func (m *ServiceMux) sendCloseAck(service uint64, closeID uint64) {
	if m.config.Output == nil {
		return
	}

	frame := make([]byte, 1+8)
	frame[0] = serviceFrameCloseAck
	binary.BigEndian.PutUint64(frame[1:9], closeID)
	if err := m.config.Output(service, frame); err != nil {
		m.reportOutputError(service, err)
	}
}

func (m *ServiceMux) registerCloseAck(service uint64, closeID uint64) chan struct{} {
	token := closeToken{service: service, id: closeID}
	ch := make(chan struct{})

	m.closeAckMu.Lock()
	m.closeAckWaiters[token] = ch
	m.closeAckMu.Unlock()

	return ch
}

func (m *ServiceMux) unregisterCloseAck(service uint64, closeID uint64) {
	token := closeToken{service: service, id: closeID}
	m.closeAckMu.Lock()
	delete(m.closeAckWaiters, token)
	m.closeAckMu.Unlock()
}

func (m *ServiceMux) notifyCloseAck(service uint64, closeID uint64) {
	token := closeToken{service: service, id: closeID}

	m.closeAckMu.Lock()
	ch, ok := m.closeAckWaiters[token]
	if ok {
		delete(m.closeAckWaiters, token)
	}
	m.closeAckMu.Unlock()

	if ok {
		close(ch)
	}
}

func (m *ServiceMux) clearCloseAckWaiters() {
	m.closeAckMu.Lock()
	defer m.closeAckMu.Unlock()

	for token, ch := range m.closeAckWaiters {
		close(ch)
		delete(m.closeAckWaiters, token)
	}
}

// NumServices returns the number of active services.
func (m *ServiceMux) NumServices() int {
	m.servicesMu.RLock()
	defer m.servicesMu.RUnlock()
	return len(m.services)
}

// NumStreams returns the total number of active yamux streams across all services.
func (m *ServiceMux) NumStreams() int {
	m.servicesMu.RLock()
	defer m.servicesMu.RUnlock()
	total := 0
	for _, e := range m.services {
		total += e.session.NumStreams()
	}
	return total
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

// getOrCreateService returns an existing service entry or creates a new one.
func (m *ServiceMux) getOrCreateService(service uint64) (*serviceEntry, error) {
	if m.closing.Load() {
		return nil, ErrServiceMuxClosed
	}

	m.servicesMu.RLock()
	entry, ok := m.services[service]
	m.servicesMu.RUnlock()
	if ok {
		return entry, nil
	}

	m.servicesMu.Lock()
	defer m.servicesMu.Unlock()

	// Double-check after acquiring write lock.
	if entry, ok := m.services[service]; ok {
		return entry, nil
	}

	if m.closed.Load() || m.closing.Load() {
		return nil, ErrServiceMuxClosed
	}

	if m.config.OnNewService != nil && !m.config.OnNewService(service) {
		return nil, ErrServiceRejected
	}
	if m.closed.Load() || m.closing.Load() {
		return nil, ErrServiceMuxClosed
	}

	entry, err := m.createServiceLocked(service)
	if err != nil {
		return nil, err
	}

	// Close 可能在 createServiceLocked 执行期间并发发生。
	// 这里二次检查，避免关闭期间把新 service 放入表中。
	if m.closed.Load() || m.closing.Load() {
		entry.session.Close()
		entry.pipe.Close()
		entry.conn.Close()
		return nil, ErrServiceMuxClosed
	}

	m.services[service] = entry
	return entry, nil
}

// createServiceLocked creates a new service entry. Must hold servicesMu write lock.
func (m *ServiceMux) createServiceLocked(service uint64) (*serviceEntry, error) {
	conn := NewKCPConn(uint32(service), func(data []byte) {
		if m.config.Output == nil {
			return
		}
		framed := make([]byte, 1+len(data))
		framed[0] = serviceFrameData
		copy(framed[1:], data)
		if err := m.config.Output(service, framed); err != nil {
			m.reportOutputError(service, err)
		}
	})

	pipe := newKCPPipe(conn)

	var session *yamux.Session
	var err error
	if m.config.IsClient {
		session, err = yamux.Client(pipe, m.config.YamuxConfig)
	} else {
		session, err = yamux.Server(pipe, m.config.YamuxConfig)
	}
	if err != nil {
		pipe.Close()
		conn.Close()
		return nil, err
	}

	entry := &serviceEntry{
		conn:    conn,
		session: session,
		pipe:    pipe,
	}

	// Start accept loop for this service's yamux session.
	m.acceptWg.Add(1)
	go m.serviceAcceptLoop(service, session)

	return entry, nil
}

// serviceAcceptLoop accepts yamux streams and forwards them to the global accept queue.
func (m *ServiceMux) serviceAcceptLoop(service uint64, session *yamux.Session) {
	defer m.acceptWg.Done()
	for {
		stream, err := session.Accept()
		if err != nil {
			return
		}

		select {
		case m.acceptCh <- acceptResult{conn: stream, service: service}:
		case <-m.closeCh:
			stream.Close()
			return
		}
	}
}

// kcpPipe adapts KCPConn to net.Conn for yamux.
// Forwards deadline calls to the underlying KCPConn so yamux's
// keepalive and timeout detection work correctly.
type kcpPipe struct {
	conn *KCPConn
}

func newKCPPipe(conn *KCPConn) *kcpPipe {
	return &kcpPipe{conn: conn}
}

func (p *kcpPipe) Read(b []byte) (int, error)         { return p.conn.Read(b) }
func (p *kcpPipe) Write(b []byte) (int, error)        { return p.conn.Write(b) }
func (p *kcpPipe) Close() error                       { return p.conn.Close() }
func (p *kcpPipe) LocalAddr() net.Addr                { return pipeAddr{} }
func (p *kcpPipe) RemoteAddr() net.Addr               { return pipeAddr{} }
func (p *kcpPipe) SetDeadline(t time.Time) error      { return p.conn.SetDeadline(t) }
func (p *kcpPipe) SetReadDeadline(t time.Time) error  { return p.conn.SetReadDeadline(t) }
func (p *kcpPipe) SetWriteDeadline(t time.Time) error { return p.conn.SetWriteDeadline(t) }

type pipeAddr struct{}

func (pipeAddr) Network() string { return "kcp" }
func (pipeAddr) String() string  { return "kcp-pipe" }

var _ net.Conn = (*kcpPipe)(nil)
