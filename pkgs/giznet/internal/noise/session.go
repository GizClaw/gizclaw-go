package noise

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// SessionState represents the current state of a session.
type SessionState uint32

const (
	// SessionStateHandshaking indicates the session is in the handshake phase.
	SessionStateHandshaking SessionState = iota
	// SessionStateEstablished indicates the session is ready for transport.
	SessionStateEstablished
	// SessionStateExpired indicates the session has expired and should not be used.
	SessionStateExpired
)

func (s SessionState) String() string {
	switch s {
	case SessionStateHandshaking:
		return "handshaking"
	case SessionStateEstablished:
		return "established"
	case SessionStateExpired:
		return "expired"
	default:
		return "unknown"
	}
}

const (
	// SessionTimeout is the duration after which a session expires without activity.
	SessionTimeout = 180 * time.Second

	// RekeyAfterMessages is the number of messages after which rekey should be triggered.
	RekeyAfterMessages = 1 << 60 // ~1 quintillion messages

	// MaxNonce is the maximum nonce value before the session must be rekeyed.
	MaxNonce = ^uint64(0) - 1 // Leave room to detect overflow
)

// Session represents an established Noise session with a peer.
// It manages the transport keys, nonces, and replay protection.
type Session struct {
	mu sync.RWMutex

	// Indices for message routing
	localIndex  uint32 // Our sender index
	remoteIndex uint32 // Peer's sender index

	// Transport keys and ciphers
	sendKey    Key
	recvKey    Key
	cipherMode CipherMode
	sendAEAD   cipher.AEAD
	recvAEAD   cipher.AEAD

	// Nonce management
	sendNonce atomic.Uint64
	recvNonce *ReplayFilter

	// State
	state    SessionState
	remotePK PublicKey // Remote peer's static public key

	// Timestamps
	createdAt    time.Time
	lastReceived time.Time
	lastSent     time.Time
}

// SessionConfig contains the configuration for creating a new session.
type SessionConfig struct {
	LocalIndex  uint32
	RemoteIndex uint32
	SendKey     Key
	RecvKey     Key
	RemotePK    PublicKey
	CipherMode  CipherMode
}

// NewSession creates a new established session from handshake results.
func NewSession(cfg SessionConfig) (*Session, error) {
	mode, err := NormalizeCipherMode(cfg.CipherMode)
	if err != nil {
		return nil, err
	}
	var sendAEAD cipher.AEAD
	var recvAEAD cipher.AEAD
	if mode != CipherModePlaintext {
		sendAEAD, err = NewAEADWithMode(mode, cfg.SendKey[:])
		if err != nil {
			return nil, err
		}
		recvAEAD, err = NewAEADWithMode(mode, cfg.RecvKey[:])
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	s := &Session{
		localIndex:   cfg.LocalIndex,
		remoteIndex:  cfg.RemoteIndex,
		sendKey:      cfg.SendKey,
		recvKey:      cfg.RecvKey,
		cipherMode:   mode,
		sendAEAD:     sendAEAD,
		recvAEAD:     recvAEAD,
		recvNonce:    NewReplayFilter(),
		state:        SessionStateEstablished,
		remotePK:     cfg.RemotePK,
		createdAt:    now,
		lastReceived: now,
		lastSent:     now,
	}

	return s, nil
}

// LocalIndex returns the local sender index.
func (s *Session) LocalIndex() uint32 {
	return s.localIndex
}

// RemoteIndex returns the remote sender index.
func (s *Session) RemoteIndex() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.remoteIndex
}

// SetRemoteIndex sets the remote sender index (during handshake).
func (s *Session) SetRemoteIndex(idx uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteIndex = idx
}

// RemotePublicKey returns the remote peer's static public key.
func (s *Session) RemotePublicKey() PublicKey {
	return s.remotePK
}

// State returns the current session state.
func (s *Session) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// SetState updates the session state.
func (s *Session) SetState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

// Encrypt encrypts a plaintext message for transport.
// Returns the ciphertext and the nonce used.
// The ciphertext includes the 16-byte authentication tag.
func (s *Session) Encrypt(plaintext []byte) (ciphertext []byte, nonce uint64, err error) {
	// Use read lock to get state and cipher (allows concurrent encryptions)
	s.mu.RLock()
	state := s.state
	cipherMode := s.cipherMode
	sendAEAD := s.sendAEAD
	s.mu.RUnlock()

	if state != SessionStateEstablished {
		return nil, 0, ErrSessionNotEstablished
	}

	// Get and increment nonce atomically
	nonce = s.sendNonce.Add(1) - 1

	// Check for nonce exhaustion
	if nonce >= MaxNonce {
		return nil, 0, ErrNonceExhausted
	}

	ciphertext, err = sealWithAEAD(cipherMode, sendAEAD, nonce, plaintext, nil)
	if err != nil {
		return nil, 0, err
	}

	// Update last sent time with write lock
	s.mu.Lock()
	s.lastSent = time.Now()
	s.mu.Unlock()

	return ciphertext, nonce, nil
}

// Decrypt decrypts a ciphertext message from transport.
// The nonce must be provided separately (from the packet header).
// Returns the plaintext if successful.
// Note: A corrupted packet that fails decryption will still consume its nonce
// in the replay filter. This is an acceptable trade-off for concurrent performance.
func (s *Session) Decrypt(ciphertext []byte, nonce uint64) ([]byte, error) {
	// Use read lock to get state and cipher (allows concurrent decryptions)
	s.mu.RLock()
	state := s.state
	cipherMode := s.cipherMode
	recvAEAD := s.recvAEAD
	s.mu.RUnlock()

	if state != SessionStateEstablished {
		return nil, ErrSessionNotEstablished
	}

	// Atomically check and update replay filter
	if !s.recvNonce.CheckAndUpdate(nonce) {
		return nil, ErrReplayDetected
	}

	plaintext, err := openWithAEAD(cipherMode, recvAEAD, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	// Update last received time with write lock
	s.mu.Lock()
	s.lastReceived = time.Now()
	s.mu.Unlock()

	return plaintext, nil
}

// IsExpired checks if the session has expired due to inactivity.
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state == SessionStateExpired {
		return true
	}

	return time.Since(s.lastReceived) > SessionTimeout
}

// Expire marks the session as expired.
func (s *Session) Expire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = SessionStateExpired
}

// CreatedAt returns when the session was created.
func (s *Session) CreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.createdAt
}

// LastReceived returns when the last message was received.
func (s *Session) LastReceived() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastReceived
}

// LastSent returns when the last message was sent.
func (s *Session) LastSent() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSent
}

// SendNonce returns the current send nonce value.
func (s *Session) SendNonce() uint64 {
	return s.sendNonce.Load()
}

// RecvMaxNonce returns the highest received nonce.
func (s *Session) RecvMaxNonce() uint64 {
	return s.recvNonce.MaxNonce()
}

// CipherMode returns the cipher mode used by the session.
func (s *Session) CipherMode() CipherMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cipherMode
}

// Errors
var (
	ErrSessionNotEstablished = errors.New("noise: session not established")
	ErrReplayDetected        = errors.New("noise: replay detected")
	ErrNonceExhausted        = errors.New("noise: nonce exhausted, rekey required")
)

// GenerateIndex generates a random 32-bit index for session identification.
func GenerateIndex() (uint32, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}
