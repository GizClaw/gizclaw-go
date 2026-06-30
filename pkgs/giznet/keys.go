package giznet

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkgs/encoding/base58"
	"golang.org/x/crypto/curve25519"
)

const KeySize = 32

type Key [KeySize]byte

type PublicKey = Key

type KeyPair struct {
	Private Key
	Public  PublicKey
}

func GenerateKeyPair() (*KeyPair, error) {
	return GenerateKeyPairFrom(rand.Reader)
}

func GenerateKeyPairFrom(random io.Reader) (*KeyPair, error) {
	var priv Key
	if _, err := io.ReadFull(random, priv[:]); err != nil {
		return nil, fmt.Errorf("giznet: failed to generate random key: %w", err)
	}
	return NewKeyPair(priv)
}

func NewKeyPair(private Key) (*KeyPair, error) {
	priv := private
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("giznet: failed to derive public key: %w", err)
	}

	kp := &KeyPair{Private: priv}
	copy(kp.Public[:], pub)
	return kp, nil
}

var ErrInvalidPublicKey = errors.New("giznet: invalid public key")

func (kp *KeyPair) DH(peerPublic PublicKey) (Key, error) {
	if kp == nil {
		return Key{}, errors.New("giznet: nil key pair")
	}
	shared, err := curve25519.X25519(kp.Private[:], peerPublic[:])
	if err != nil {
		return Key{}, fmt.Errorf("giznet: DH failed: %w", ErrInvalidPublicKey)
	}

	var result Key
	copy(result[:], shared)
	if result.IsZero() {
		return Key{}, ErrInvalidPublicKey
	}
	return result, nil
}

func KeyFromHex(s string) (Key, error) {
	var k Key
	b, err := hex.DecodeString(s)
	if err != nil {
		return k, fmt.Errorf("giznet: invalid hex string: %w", err)
	}
	if len(b) != KeySize {
		return k, fmt.Errorf("giznet: invalid key length: got %d, want %d", len(b), KeySize)
	}
	copy(k[:], b)
	return k, nil
}

func (k Key) IsZero() bool {
	var zero Key
	return k == zero
}

func (k Key) String() string {
	text, _ := k.MarshalText()
	return string(text)
}

func (k Key) ShortString() string {
	return hex.EncodeToString(k[:4])
}

func (k Key) MarshalText() ([]byte, error) {
	return []byte(base58.EncodeToString(k[:])), nil
}

func (k *Key) UnmarshalText(text []byte) error {
	if k == nil {
		return errors.New("giznet: nil key")
	}
	decoded, err := decodeKeyText(text)
	if err != nil {
		return err
	}
	copy(k[:], decoded)
	return nil
}

func (k Key) Equal(other Key) bool {
	var result byte
	for i := range KeySize {
		result |= k[i] ^ other[i]
	}
	return result == 0
}

func decodeKeyText(text []byte) ([]byte, error) {
	value := strings.TrimSpace(string(text))
	if value == "" {
		return nil, errors.New("giznet: empty key")
	}
	decoded, err := base58.DecodeString(value)
	if err != nil || len(decoded) != KeySize {
		return nil, fmt.Errorf("giznet: invalid key text")
	}
	return decoded, nil
}
