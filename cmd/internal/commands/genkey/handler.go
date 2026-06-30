package genkey

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/gizrun"
)

type Handler struct {
	Out io.Writer
}

func (h Handler) Execute(context.Context, gizrun.CommandLine) error {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.out(), keyPair.Private.String())
	return err
}

func (h Handler) out() io.Writer {
	if h.Out != nil {
		return h.Out
	}
	return os.Stdout
}
