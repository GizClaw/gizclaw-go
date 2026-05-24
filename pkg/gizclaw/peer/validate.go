package peer

import (
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

func validateGear(gear apitypes.Gear) error {
	if key, err := publicKeyFromText(gear.PublicKey); err != nil {
		return err
	} else if key.IsZero() {
		return fmt.Errorf("gear: empty public key")
	}
	if !gear.Role.Valid() {
		return fmt.Errorf("gear: invalid role %q", gear.Role)
	}
	if !gear.Status.Valid() {
		return fmt.Errorf("gear: invalid status %q", gear.Status)
	}
	return validateConfiguration(gear.Configuration)
}

func validateConfiguration(apitypes.Configuration) error {
	return nil
}

func publicKeyFromText(publicKey string) (giznet.PublicKey, error) {
	var key giznet.PublicKey
	if err := key.UnmarshalText([]byte(publicKey)); err != nil {
		return giznet.PublicKey{}, fmt.Errorf("gear: invalid public key: %w", err)
	}
	return key, nil
}
