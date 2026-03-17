package main

import (
	"os"

	"github.com/labtether/labtether-cli/cmd"
)

func main() {
	code := cmd.Execute()
	os.Exit(code)
}
