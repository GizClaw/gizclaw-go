package main

import (
	"os"

	"github.com/giztoy/giztoy-go/cmd/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
