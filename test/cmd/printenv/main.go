//go:build e2e
// +build e2e

package main

import (
	"fmt"
	"os"

	"github.com/moreh-dev/mif/test/utils"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "env" {
		utils.PrintEnvVarsHelp()
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "Usage: go run -tags=e2e ./test/cmd/printenv env\n")
	os.Exit(1)
}
