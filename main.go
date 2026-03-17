package main

import (
	"os"

	"github.com/labtether/labtether-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
