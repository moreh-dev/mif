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

// DeleteGatewayResources deletes Gateway resources (Gateway, ConfigMap, or GatewayParameters) from the given namespace.
func DeleteGatewayResources(workloadNamespace, gatewayClass string) error {
	By("deleting Gateway resource")
	cmd := exec.Command("kubectl", "delete", "gateway", "mif",
		"-n", workloadNamespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(fmt.Errorf("failed to delete Gateway: %w", err))
	}

	switch gatewayClass {
	case "istio":
		By("deleting Gateway infrastructure ConfigMap")
		cmd := exec.Command("kubectl", "delete", "configmap", "mif-gateway-infrastructure",
			"-n", workloadNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(fmt.Errorf("failed to delete Gateway ConfigMap: %w", err))
		}
	case "kgateway":
		By("deleting Gateway infrastructure GatewayParameters")
		cmd := exec.Command("kubectl", "delete", "gatewayparameters", "mif-gateway-infrastructure",
			"-n", workloadNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(fmt.Errorf("failed to delete Gateway GatewayParameters: %w", err))
		}
	}

	return nil
}

// CleanupWorkloadNamespace cleans up test resources in the workload namespace.
// It deletes Gateway resources, InferenceService, uninstalls Heimdall, and deletes the namespace in order.
// If mifNamespace equals workloadNamespace, the namespace deletion is skipped to avoid deleting MIF infrastructure.
func CleanupWorkloadNamespace(workloadNamespace, inferenceServiceName, gatewayClass, mifNamespace string) error {
	if err := DeleteInferenceService(workloadNamespace, inferenceServiceName); err != nil {
		warnError(fmt.Errorf("failed to delete InferenceService: %w", err))
	}

	if err := UninstallHeimdall(workloadNamespace); err != nil {
		warnError(fmt.Errorf("failed to uninstall Heimdall: %w", err))
	}

	if err := DeleteGatewayResources(workloadNamespace, gatewayClass); err != nil {
		warnError(fmt.Errorf("failed to delete Gateway resources: %w", err))
	}

	if mifNamespace != workloadNamespace {
		if err := DeleteNamespace(workloadNamespace); err != nil {
			return fmt.Errorf("failed to delete workload namespace: %w", err)
		}
	} else {
		By(fmt.Sprintf("skipping namespace deletion: workloadNamespace (%s) is the same as mifNamespace", workloadNamespace))
	}

	return nil
}
