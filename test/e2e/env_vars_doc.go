//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os"
)

// envVars, getUsedEnvVars, and validateEnvVars are defined in env_vars.go

// PrintEnvVarsHelp prints documentation for E2E test environment variables.
func PrintEnvVarsHelp() {
	// Validate that all used env vars are documented
	if err := validateEnvVars(); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: %v\n\n", err)
	}

	fmt.Println("E2E Test Environment Variables:")
	fmt.Println()

	var currentCategory string
	for _, env := range envVars {
		if env.Category != currentCategory {
			if currentCategory != "" {
				fmt.Println()
			}
			fmt.Printf("%s:\n", env.Category)
			currentCategory = env.Category
		}

		defaultStr := env.DefaultValue
		if defaultStr != "" && env.Type != "optional" {
			defaultStr = fmt.Sprintf("(default: %s)", defaultStr)
		} else if env.Type == "optional" {
			defaultStr = env.DefaultValue
		}

		if defaultStr != "" {
			fmt.Printf("  %-35s %s %s\n", env.Name, env.Description, defaultStr)
		} else {
			fmt.Printf("  %-35s %s\n", env.Name, env.Description)
		}
	}

	fmt.Println()
	fmt.Println("Inference Image defaults:")
	fmt.Println("  If not set, uses default based on cluster type:")
	fmt.Println("  - kind: ghcr.io/llm-d/llm-d-inference-sim:v0.6.1")
	fmt.Println("  - kubeconfig: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm:20250915.1")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  SKIP_KIND=true SKIP_PREREQUISITE=true make test-e2e")
}
