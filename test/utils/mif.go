//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/moreh-dev/mif/test/utils/settings"
)

func renderMIFValues(awsAccessKeyID, awsSecretAccessKey string) (string, error) {
	data := struct {
		AWSAccessKeyID     string
		AWSSecretAccessKey string
	}{
		AWSAccessKeyID:     awsAccessKeyID,
		AWSSecretAccessKey: awsSecretAccessKey,
	}

	return RenderTemplate(settings.MIFValuesTemplate, data)
}

// InstallMIF installs MIF infrastructure if not already installed.
func InstallMIF(namespace string, awsAccessKeyID, awsSecretAccessKey string) error {
	if awsAccessKeyID == "" || awsSecretAccessKey == "" {
		return fmt.Errorf("AWS access key ID and secret access key are required")
	}

	values, err := renderMIFValues(awsAccessKeyID, awsSecretAccessKey)
	if err != nil {
		return fmt.Errorf("failed to create moai-inference-framework values file: %w", err)
	}

	helmArgs := []string{
		"upgrade", "--install", "mif",
		"deploy/helm/moai-inference-framework",
		"--namespace", namespace,
		"--create-namespace",
		"--wait",
		"-f", "-",
		fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong),
	}

	cmd := exec.Command("helm", helmArgs...)
	cmd.Stdin = strings.NewReader(values)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to deploy MIF via Helm: %w", err)
	}
	return nil
}

// UninstallMIF uninstalls MIF from the given namespace.
func UninstallMIF(namespace string) {
	cmd := exec.Command("helm", "uninstall", "mif", "-n", namespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallMIFPreset deploys moai-inference-preset in the given namespace.
func InstallMIFPreset(namespace string) error {
	helmArgs := []string{
		"upgrade", "--install", "moai-inference-preset",
		"deploy/helm/moai-inference-preset",
		"--namespace", namespace,
		"--create-namespace",
		"--wait",
		fmt.Sprintf("--timeout=%v", settings.TimeoutLong),
	}

	cmd := exec.Command("helm", helmArgs...)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to deploy MIF preset via Helm: %w", err)
	}
	return nil
}

// UninstallMIFPreset uninstalls moai-inference-preset from the given namespace.
func UninstallMIFPreset(namespace string) {
	cmd := exec.Command("helm", "uninstall", "moai-inference-preset", "-n", namespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}
