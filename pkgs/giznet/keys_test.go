package giznet

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestKeyPairGenerationAndDH(t *testing.T) {
	alice, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(alice) error = %v", err)
	}
	bob, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(bob) error = %v", err)
	}
	if alice.Private.IsZero() || alice.Public.IsZero() {
		t.Fatal("alice key pair contains zero key")
	}
	if alice.Public == bob.Public {
		t.Fatal("generated public keys are equal")
	}

	aliceShared, err := alice.DH(bob.Public)
	if err != nil {
		t.Fatalf("alice DH error = %v", err)
	}
	bobShared, err := bob.DH(alice.Public)
	if err != nil {
		t.Fatalf("bob DH error = %v", err)
	}
	if aliceShared != bobShared {
		t.Fatal("DH shared keys differ")
	}
}

func TestKeyPairFromPrivateClampsAndDerivesPublicKey(t *testing.T) {
	var private Key
	for i := range private {
		private[i] = byte(i + 1)
	}
	kp, err := NewKeyPair(private)
	if err != nil {
		t.Fatalf("NewKeyPair error = %v", err)
	}
	if kp.Private == private {
		t.Fatal("private key was not clamped")
	}
	if kp.Private[0]&7 != 0 || kp.Private[31]&0x80 != 0 || kp.Private[31]&0x40 == 0 {
		t.Fatalf("private key was not clamped correctly: first=%08b last=%08b", kp.Private[0], kp.Private[31])
	}
	if kp.Public.IsZero() {
		t.Fatal("derived public key is zero")
	}
}

func TestKeyTextEncoding(t *testing.T) {
	key, err := KeyFromHex(strings.Repeat("01", KeySize))
	if err != nil {
		t.Fatalf("KeyFromHex error = %v", err)
	}
	text, err := key.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	if got := key.String(); got != string(text) {
		t.Fatalf("String() = %q, want %q", got, string(text))
	}
	if got := key.ShortString(); got != "01010101" {
		t.Fatalf("ShortString() = %q", got)
	}

	var decoded Key
	if err := decoded.UnmarshalText(bytes.TrimSpace(text)); err != nil {
		t.Fatalf("UnmarshalText error = %v", err)
	}
	if !decoded.Equal(key) {
		t.Fatalf("decoded key = %v, want %v", decoded, key)
	}
}

func TestKeyErrors(t *testing.T) {
	if _, err := KeyFromHex("bad"); err == nil {
		t.Fatal("KeyFromHex accepted invalid hex")
	}
	if _, err := KeyFromHex("01"); err == nil {
		t.Fatal("KeyFromHex accepted short key")
	}
	var nilKey *Key
	if err := nilKey.UnmarshalText([]byte("x")); err == nil {
		t.Fatal("nil key UnmarshalText succeeded")
	}
	var key Key
	if err := key.UnmarshalText(nil); err == nil {
		t.Fatal("empty key UnmarshalText succeeded")
	}
	if err := key.UnmarshalText([]byte("not-base58-key")); err == nil {
		t.Fatal("invalid key text UnmarshalText succeeded")
	}
	if _, err := (*KeyPair)(nil).DH(PublicKey{}); err == nil {
		t.Fatal("nil key pair DH succeeded")
	}
	kp, err := NewKeyPair(Key{1})
	if err != nil {
		t.Fatalf("NewKeyPair error = %v", err)
	}
	if _, err := kp.DH(PublicKey{}); !errors.Is(err, ErrInvalidPublicKey) {
		t.Fatalf("DH zero peer error = %v, want %v", err, ErrInvalidPublicKey)
	}
}

func TestGenerateKeyPairFromReaderError(t *testing.T) {
	_, err := GenerateKeyPairFrom(bytes.NewReader([]byte{1, 2, 3}))
	if err == nil {
		t.Fatal("GenerateKeyPairFrom short reader succeeded")
	}
}
