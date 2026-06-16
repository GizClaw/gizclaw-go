package main

import (
	"io"
	"os"

	"github.com/GizClaw/gizclaw-go/cmd/internal/commands"
)

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		os.Exit(1)
	}
}

func run(args []string, stderr io.Writer) error {
	_ = stderr
	return commands.New().Execute()
}
