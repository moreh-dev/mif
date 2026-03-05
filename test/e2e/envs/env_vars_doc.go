//go:build e2e
// +build e2e

package envs

import (
	"fmt"
	"os"
)

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
	fmt.Println("Note: Model names, template refs, S3 region/bucket, and other fixed")
	fmt.Println("configuration values are hardcoded in test code and settings/constants.go.")
	fmt.Println("Only execution settings (SKIP_*), credentials, and environment-specific")
	fmt.Println("values (WORKLOAD_NAMESPACE, ISTIO_REV) are configurable via env vars.")
	fmt.Println()
	fmt.Println("Example (product cluster with kubeconfig):")
	fmt.Println("  SKIP_PREREQUISITE=true ISTIO_REV=1-28-1 make test-e2e-performance")
	fmt.Println()
	fmt.Println("Example (local Kind cluster with defaults):")
	fmt.Println("  make test-e2e-kind")
}
