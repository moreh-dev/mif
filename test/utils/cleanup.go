package utils

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
)

// DeleteNamespace deletes a namespace, with fallback to force delete by removing finalizers if needed.
func DeleteNamespace(namespace string) error {
	By(fmt.Sprintf("deleting namespace %s", namespace))
	cmd := exec.Command("kubectl", "delete", "ns", namespace, "--timeout=60s", "--ignore-not-found=true")
	output, err := Run(cmd)
	if err != nil {
		By("attempting to force delete namespace by removing finalizers")
		cmd = exec.Command("kubectl", "patch", "namespace", namespace,
			"--type=json", "-p", `[{"op": "replace", "path": "/spec/finalizers", "value": []}]`, "--ignore-not-found=true")
		_, _ = Run(cmd)

		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--timeout=30s", "--ignore-not-found=true")
		_, err = Run(cmd)
		if err != nil {
			return fmt.Errorf("failed to delete namespace: %w", err)
		}
	} else if output != "" {
		_, _ = fmt.Fprintf(GinkgoWriter, "Namespace deletion output: %s\n", output)
	}

	return nil
}

// CleanupWorkloadNamespace cleans up test resources in the workload namespace.
// It deletes InferenceService, uninstalls Heimdall, and deletes the namespace in order.
func CleanupWorkloadNamespace(workloadNamespace, inferenceServiceName string) error {
	if err := DeleteInferenceService(workloadNamespace, inferenceServiceName); err != nil {
		warnError(fmt.Errorf("failed to delete InferenceService: %w", err))
	}

	if err := UninstallHeimdall(workloadNamespace); err != nil {
		warnError(fmt.Errorf("failed to uninstall Heimdall: %w", err))
	}

	if err := DeleteNamespace(workloadNamespace); err != nil {
		return fmt.Errorf("failed to delete workload namespace: %w", err)
	}

	return nil
}
