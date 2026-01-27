//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/moreh-dev/mif/test/utils/settings"
)

func renderHeimdallValues(gatewayClassName string) (string, error) {
	data := struct {
		MorehRegistrySecretName string
		GatewayName             string
		GatewayClass            string
	}{
		MorehRegistrySecretName: settings.MorehRegistrySecretName,
		GatewayName:             settings.GatewayName,
		GatewayClass:            gatewayClassName,
	}

	return RenderTemplate(settings.HeimdallValuesTemplate, data)
}

// InstallHeimdall installs Heimdall in the given namespace.
func InstallHeimdall(namespace string, gatewayClassName string) error {
	cmd := exec.Command("helm", "repo", "add", "moreh", settings.MorehHelmRepoURL)
	if _, err := Run(cmd); err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to add moreh helm repo: %w", err)
	}

	cmd = exec.Command("helm", "repo", "update", "moreh")
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to update moreh helm repo: %w", err)
	}

	values, err := renderHeimdallValues(gatewayClassName)
	if err != nil {
		return fmt.Errorf("failed to create Heimdall values file: %w", err)
	}

	helmArgs := []string{
		"upgrade", "--install", "heimdall",
		"moreh/heimdall",
		"--version", settings.HeimdallVersion,
		"--namespace", namespace,
		"--create-namespace",
		"--wait",
		"-f", "-",
		fmt.Sprintf("--timeout=%v", settings.TimeoutLong),
	}

	cmd = exec.Command("helm", helmArgs...)
	cmd.Stdin = strings.NewReader(values)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to install Heimdall: %w", err)
	}
	return nil
}

// UninstallHeimdall uninstalls Heimdall from the given namespace.
func UninstallHeimdall(namespace string) {
	cmd := exec.Command("helm", "uninstall", "heimdall", "-n", namespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}
