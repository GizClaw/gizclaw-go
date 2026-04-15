package firmware

import "testing"

func TestChannelValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value Channel
		valid bool
	}{
		{name: "beta", value: Beta, valid: true},
		{name: "rollback", value: Rollback, valid: true},
		{name: "stable", value: Stable, valid: true},
		{name: "testing", value: Testing, valid: true},
		{name: "invalid", value: Channel("dev"), valid: false},
		{name: "empty", value: "", valid: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.value.Valid(); got != tc.valid {
				t.Fatalf("Valid(%q) = %v, want %v", tc.value, got, tc.valid)
			}
		})
	}
}
