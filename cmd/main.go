package main

import (
	"os"

	"github.com/giztoy/giztoy-go/cmd/internal/commands"
)

func main() {
	if err := commands.New().Execute(); err != nil {
		os.Exit(1)
	}
}
