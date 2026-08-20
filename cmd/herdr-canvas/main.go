package main

import (
	"errors"
	"os"

	"herdr-canvas/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		var code interface{ ExitCode() int }
		if errors.As(err, &code) {
			os.Exit(code.ExitCode())
		}
		os.Exit(1)
	}
}
