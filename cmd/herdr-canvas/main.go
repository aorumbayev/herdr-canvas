package main

import (
	"os"

	"herdr-canvas/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
