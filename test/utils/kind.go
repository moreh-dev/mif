//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/moreh-dev/mif/test/utils/settings"
)

// CreateKindCluster creates a kind cluster with the given name.
func CreateKindCluster() error {
	args := []string{"create", "cluster", "--name", settings.KindClusterName}

	cmd := exec.Command("kind", args...)
	if err := RunWithGinkgoWriter(cmd); err != nil {
		return fmt.Errorf("failed to create kind cluster: %w", err)
	}
	return nil
}

// DeleteKindCluster deletes a kind cluster with the given name.
func DeleteKindCluster() {
	cmd := exec.Command("kind", "delete", "cluster", "--name", settings.KindClusterName)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// IsKindClusterExists checks if a kind cluster with the given name exists.
func IsKindClusterExists() bool {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	for _, name := range strings.Fields(output) {
		if name == settings.KindClusterName {
			return true
		}
	}
	return false
}
