package giznoise

import (
	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/internal/noise"
)

func toNoiseKey(k giznet.Key) noise.Key {
	var out noise.Key
	copy(out[:], k[:])
	return out
}

func toNoisePublicKey(k giznet.PublicKey) noise.PublicKey {
	var out noise.PublicKey
	copy(out[:], k[:])
	return out
}

func fromNoisePublicKey(k noise.PublicKey) giznet.PublicKey {
	var out giznet.PublicKey
	copy(out[:], k[:])
	return out
}

func toNoiseKeyPair(k *giznet.KeyPair) *noise.KeyPair {
	if k == nil {
		return nil
	}
	return &noise.KeyPair{
		Private: toNoiseKey(k.Private),
		Public:  toNoisePublicKey(k.Public),
	}
}
