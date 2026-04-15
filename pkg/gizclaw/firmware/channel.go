package firmware

// Channel defines firmware release channels used by firmware.
type Channel string

// Defines values for Channel.
const (
	Beta     Channel = "beta"
	Rollback Channel = "rollback"
	Stable   Channel = "stable"
	Testing  Channel = "testing"
)

// Valid indicates whether the value is a known member of the Channel enum.
func (e Channel) Valid() bool {
	switch e {
	case Beta:
		return true
	case Rollback:
		return true
	case Stable:
		return true
	case Testing:
		return true
	default:
		return false
	}
}
