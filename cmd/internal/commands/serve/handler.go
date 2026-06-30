package servecmd

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/GizClaw/gizclaw-go/cmd/internal/server"
	"github.com/GizClaw/gizclaw-go/pkgs/gizrun"
)

type Handler struct{}

func (Handler) Execute(_ context.Context, commandLine gizrun.CommandLine) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	var force bool
	fs.BoolVar(&force, "force", false, "explicitly allow foreground local serve and replace stale pid files")
	fs.BoolVar(&force, "f", false, "explicitly allow foreground local serve and replace stale pid files")
	if err := fs.Parse(commandLine.Flags); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("serve: unexpected flags: %v", fs.Args())
	}
	if len(commandLine.Args) != 2 {
		return errors.New("serve: expected workspace dir")
	}
	return server.ServeWithOptions(commandLine.Args[1], server.ServeOptions{
		Force: force,
	})
}
