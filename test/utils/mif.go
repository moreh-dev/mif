package utils

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
)

// CreateInferenceService creates an InferenceService CR in the given namespace.
func CreateInferenceService(namespace, valuesPath string) error {
	if valuesPath == "" {
		return fmt.Errorf("inference service manifest path is required (e.g., path/to/manifest.yaml)")
	}

	kubectlArgs := []string{
		"apply",
		"-f", valuesPath,
	}

	if namespace != "" {
		kubectlArgs = append(kubectlArgs, "-n", namespace)
	}

	cmd := exec.Command("kubectl", kubectlArgs...)
	_, err := Run(cmd)
	return err
}

// DeleteInferenceService deletes an InferenceService from the given namespace.
func DeleteInferenceService(namespace, inferenceServiceName string) error {
	By("deleting InferenceService")
	cmd := exec.Command("kubectl", "delete", "inferenceservice", inferenceServiceName,
		"-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

// InstallHeimdall installs Heimdall in the given namespace.
func InstallHeimdall(namespace string, valuesPath string) error {
	By("adding moreh Helm repository")
	cmd := exec.Command("helm", "repo", "add", "moreh", "https://moreh-dev.github.io/helm-charts")
	if _, err := Run(cmd); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to add moreh helm repo: %w", err)
		}
	}

	By("updating moreh Helm repository")
	cmd = exec.Command("helm", "repo", "update", "moreh")
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to update moreh helm repo: %w", err)
	}

	By("installing Heimdall")
	helmArgs := []string{
		"upgrade", "--install", "heimdall",
		"moreh/heimdall",
		"--version", "v0.6.0",
		"--namespace", namespace,
		"--create-namespace",
	}
	if valuesPath != "" {
		helmArgs = append(helmArgs, "-f", valuesPath)
	}
	cmd = exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

// UninstallHeimdall uninstalls Heimdall from the given namespace.
func UninstallHeimdall(namespace string) error {
	cmd := exec.Command("helm", "uninstall", "heimdall", "-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

// DeployMIFPreset deploys moai-inference-preset in the given namespace.
func DeployMIFPreset(namespace string, chartPath string) error {
	By("deploying moai-inference-preset")
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
