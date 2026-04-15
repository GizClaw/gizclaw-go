package servecmd

import "testing"

func TestServeCommandRequiresSingleWorkspaceArg(t *testing.T) {
	cmd := NewCmd()
	if err := cmd.Args(cmd, []string{"workspace-dir"}); err != nil {
		t.Fatalf("Args(valid) error = %v", err)
	}
	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("Args(nil) should fail")
	}
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Fatal("Args(two args) should fail")
	}
}
