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

// CleanupConfig holds configuration for cleaning up test resources in a workload namespace.
type CleanupConfig struct {
	WorkloadNamespace string
	GatewayClass      string
	MIFNamespace      string
	PrefillName       string
	DecodeName        string
	TemplateNames     []string
}

// CleanupWorkloadNamespace cleans up test resources in the workload namespace.
// It deletes Gateway resources, InferenceService(s), InferenceServiceTemplate(s), uninstalls Heimdall, and deletes the namespace in order.
// If MIFNamespace equals WorkloadNamespace, the namespace deletion is skipped to avoid deleting MIF infrastructure.
// If PrefillName or DecodeName is empty, the corresponding InferenceService deletion is skipped.
// If TemplateNames is empty, InferenceServiceTemplate deletion is skipped.
func CleanupWorkloadNamespace(config CleanupConfig) error {
	for _, templateName := range config.TemplateNames {
		if err := DeleteInferenceServiceTemplate(config.WorkloadNamespace, templateName); err != nil {
			warnError(fmt.Errorf("failed to delete InferenceServiceTemplate %s: %w", templateName, err))
		}
	}
	
	if config.PrefillName != "" {
		if err := DeleteInferenceService(config.WorkloadNamespace, config.PrefillName); err != nil {
			warnError(fmt.Errorf("failed to delete prefill InferenceService: %w", err))
		}
	}
	if config.DecodeName != "" {
		if err := DeleteInferenceService(config.WorkloadNamespace, config.DecodeName); err != nil {
			warnError(fmt.Errorf("failed to delete decode InferenceService: %w", err))
		}
	}

	if err := UninstallHeimdall(config.WorkloadNamespace); err != nil {
		warnError(fmt.Errorf("failed to uninstall Heimdall: %w", err))
	}

	if err := DeleteGatewayResources(config.WorkloadNamespace, config.GatewayClass); err != nil {
		warnError(fmt.Errorf("failed to delete Gateway resources: %w", err))
	}

	if config.MIFNamespace != config.WorkloadNamespace {
		if err := DeleteNamespace(config.WorkloadNamespace); err != nil {
			return fmt.Errorf("failed to delete workload namespace: %w", err)
		}
	} else {
		By(fmt.Sprintf("skipping namespace deletion: workloadNamespace (%s) is the same as mifNamespace", config.WorkloadNamespace))
	}

	return nil
}
