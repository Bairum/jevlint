package main

import (
	"context"
	"os"

	"github.com/codegirl-007/jevlint/internal/cli"
)

// main runs the jevlint command and exits with its status code.
func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}
