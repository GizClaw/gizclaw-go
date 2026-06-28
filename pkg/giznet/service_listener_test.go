package giznet

import "testing"

func TestServiceListenerContract(t *testing.T) {
	var _ ServiceListener = fakeServiceListener{}
}
