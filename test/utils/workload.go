//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateWorkloadNamespace creates the workload namespace and labels.
func CreateWorkloadNamespace(namespace string, mifNamespace string) error {
	cmd := exec.Command("kubectl", "create", "ns", namespace, "--request-timeout=30s")
	_, err := Run(cmd)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		return fmt.Errorf("failed to create workload namespace: %w", err)
	}

	if mifNamespace != namespace {
		cmd = exec.Command("kubectl", "label", "namespace", namespace,
			"mif=enabled", "--overwrite", "--request-timeout=30s")
		_, err = Run(cmd)
		if err != nil {
			return fmt.Errorf("failed to add mif=enabled label to namespace: %w", err)
		}
	}
	return nil
}
