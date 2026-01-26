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

// createMIFValuesFile creates a values file for moai-inference-framework with ECR token refresher configuration.
func createMIFValuesFile(awsAccessKeyID, awsSecretAccessKey string) (string, error) {
	valuesContent := fmt.Sprintf(`ecrTokenRefresher:
  aws:
    accessKeyId: %s
    secretAccessKey: %s

fullnameOverride: %s
`, awsAccessKeyID, awsSecretAccessKey, helmReleaseMIF)

	return writeValuesFile(tempFileMIFValues, valuesContent, 0600)
}

// setupMIF installs MIF infrastructure if not already installed.
func setupMIF() {
	By("creating MIF namespace")
	cmd := exec.Command("kubectl", "create", "ns", cfg.mifNamespace)
	_, err := Run(cmd)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		Expect(err).NotTo(HaveOccurred(), "Failed to create MIF namespace")
	}

	By("deploying MIF infrastructure via Helm")

	helmArgs := []string{
		"upgrade", "--install", helmReleaseMIF,
		cfg.mifChartPath,
		"--namespace", cfg.mifNamespace,
		"--wait",
		fmt.Sprintf("--timeout=%v", TimeoutVeryLong),
		"--set", "odin.enabled=true",
		"--set", fmt.Sprintf("odin-crd.enabled=%t", cfg.odinCRDEnabled),
		"--set", fmt.Sprintf("prometheus-stack.enabled=%t", cfg.prometheusStackEnabled),
		"--set", fmt.Sprintf("keda.enabled=%t", cfg.kedaEnabled),
		"--set", fmt.Sprintf("lws.enabled=%t", cfg.lwsEnabled),
		"--set", "replicator.enabled=true",
	}

	if cfg.awsAccessKeyID != "" && cfg.awsSecretAccessKey != "" {
		By("creating moai-inference-framework values file for ECR token refresher")
		mifValuesPath, err := createMIFValuesFile(cfg.awsAccessKeyID, cfg.awsSecretAccessKey)
		Expect(err).NotTo(HaveOccurred(), "Failed to create moai-inference-framework values file")
		helmArgs = append(helmArgs, "-f", mifValuesPath)
	}

	cmd = exec.Command("helm", helmArgs...)
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to deploy MIF via Helm")

	By("waiting for MIF components to be ready")
	waitForMIFComponents()
}

// waitForMIFComponents waits for MIF components to be ready.
func waitForMIFComponents() {
	By("waiting for Odin controller deployment")
	cmd := exec.Command("kubectl", "get", "deployment",
		"-n", cfg.mifNamespace,
		"-l", "app.kubernetes.io/name=odin",
		"-o", "jsonpath={.items[0].metadata.name}")
	output, err := Run(cmd)
	if err != nil || strings.TrimSpace(output) == "" {
		cmd = exec.Command("kubectl", "get", "deployment",
			"-n", cfg.mifNamespace,
			"-o", "jsonpath={.items[?(@.metadata.name=~\"odin.*\")].metadata.name}")
		output, err = Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Could not find Odin deployment by label, trying common name pattern\n")
			cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
				fmt.Sprintf("deployment/%s-odin", helmReleaseMIF),
				"-n", cfg.mifNamespace,
				fmt.Sprintf("--timeout=%v", TimeoutMedium))
			_, _ = Run(cmd)
			return
		}
	}
	odinDeploymentName := strings.TrimSpace(output)
	_, _ = fmt.Fprintf(GinkgoWriter, "Found Odin deployment: %s\n", odinDeploymentName)

	cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
		fmt.Sprintf("deployment/%s", odinDeploymentName),
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutLong))
	output, err = Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Odin deployment wait completed with error (may already be ready): %v\n", err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Odin deployment is available\n")
	}

	By("waiting for all running pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"--for=condition=Ready",
		"--all",
		"--field-selector=status.phase!=Succeeded",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutLong))
	output, err = Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Some pods may not be ready yet: %v\n", err)
		cmd = exec.Command("kubectl", "get", "pods",
			"-n", cfg.mifNamespace,
			"--field-selector=status.phase!=Succeeded",
			"-o", "wide")
		statusOutput, _ := Run(cmd)
		_, _ = fmt.Fprintf(GinkgoWriter, "Current pod status:\n%s\n", statusOutput)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "All pods are ready\n")
	}
}

// DeployMIFPreset deploys moai-inference-preset in the given namespace.
func DeployMIFPreset(namespace string, chartPath string) error {
	helmArgs := []string{
		"upgrade", "--install", "moai-inference-preset",
		chartPath,
		"--namespace", namespace,
		"--wait",
		"--timeout", "10m",
	}
	cmd := exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

// UninstallMIFPreset uninstalls moai-inference-preset from the given namespace.
func UninstallMIFPreset(namespace string) error {
	cmd := exec.Command("helm", "uninstall", "moai-inference-preset", "-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

func cleanupMIFNamespace() {
	if err := DeleteNamespace(cfg.mifNamespace); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to delete MIF namespace: %v\n", err)
	}
}
