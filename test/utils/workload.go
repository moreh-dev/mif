//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// CreateWorkloadNamespace creates the workload namespace and labels.
func CreateWorkloadNamespace() {
	By("creating workload namespace")
	cmd := exec.Command("kubectl", "create", "ns", cfg.workloadNamespace, "--request-timeout=30s")
	_, err := Run(cmd)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		Expect(err).NotTo(HaveOccurred(), "Failed to create workload namespace")
	}

	if cfg.mifNamespace != cfg.workloadNamespace {
		By("adding mif=enabled label to workload namespace for automatic secret copying")
		cmd = exec.Command("kubectl", "label", "namespace", cfg.workloadNamespace,
			"mif=enabled", "--overwrite", "--request-timeout=30s")
		_, err = Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add mif=enabled label to namespace: %v\n", err)
		}
	}

	if cfg.istioRev != "" {
		By(fmt.Sprintf("adding istio.io/rev=%s label to workload namespace", cfg.istioRev))
		cmd = exec.Command("kubectl", "label", "namespace", cfg.workloadNamespace,
			fmt.Sprintf("istio.io/rev=%s", cfg.istioRev), "--overwrite", "--request-timeout=30s")
		_, err = Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add istio.io/rev label to namespace: %v\n", err)
		}
	}
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
func CleanupWorkloadNamespace() error {
	config := CleanupConfig{
		WorkloadNamespace: cfg.workloadNamespace,
		GatewayClass:      cfg.gatewayClass,
		MIFNamespace:      cfg.mifNamespace,
		PrefillName:       fmt.Sprintf("%s-prefill", inferenceServiceName),
		DecodeName:        fmt.Sprintf("%s-decode", inferenceServiceName),
		TemplateNames: []string{
			"workertemplate-vllm-common",
			"workertemplate-pd-prefill-meta",
			"workertemplate-pd-decode-meta",
			"workertemplate-decode-proxy",
		},
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

	for _, templateName := range config.TemplateNames {
		if err := DeleteInferenceServiceTemplate(config.WorkloadNamespace, templateName); err != nil {
			warnError(fmt.Errorf("failed to delete InferenceServiceTemplate %s: %w", templateName, err))
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
