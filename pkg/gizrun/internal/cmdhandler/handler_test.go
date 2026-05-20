package cmdhandler

import (
	"reflect"
	"testing"
)

func TestCommandLineFromArgv(t *testing.T) {
	got := CommandLineFromArgv([]string{
		"-config",
		"app.yaml",
		"chat",
		"hello",
		"-model=gpt",
		"--",
		"-literal",
	})
	if !reflect.DeepEqual(got.Args, []string{"app.yaml", "chat", "hello", "-literal"}) {
		t.Fatalf("args = %#v, want app yaml chat hello literal", got.Args)
	}
	if !reflect.DeepEqual(got.Flags, []string{"-config", "-model=gpt"}) {
		t.Fatalf("flags = %#v, want config and model flags", got.Flags)
	}
}
