//go:build e2e
// +build e2e

package main

import (
	"fmt"
	"os"

	e2e "github.com/moreh-dev/mif/test/e2e/envs"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "env" {
		e2e.PrintEnvVarsHelp()
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "Usage: go run -tags=e2e ./test/cmd/printenv env\n")
	os.Exit(1)
}
