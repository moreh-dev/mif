//go:build printenv && e2e
// +build printenv,e2e

package main

import (
	"fmt"
	"os"

	"github.com/moreh-dev/mif/test/e2e"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "env" {
		e2e.PrintEnvVarsHelp()
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "Usage: go run -tags=printenv,e2e ./test/e2e/cmd/printenv env\n")
	os.Exit(1)
}
