package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// InstallGatewayAPI installs Gateway API.
func InstallGatewayAPI() error {
	cmd := exec.Command("kubectl", "apply", "--server-side",
		"-f", "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml")
	_, err := Run(cmd)
	return err
}

// UninstallGatewayAPI uninstalls Gateway API.
func UninstallGatewayAPI() error {
	cmd := exec.Command("kubectl", "delete",
		"-f", "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml",
		"--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

// InstallGatewayInferenceExtension installs Gateway Inference Extension.
func InstallGatewayInferenceExtension() error {
	cmd := exec.Command("kubectl", "apply",
		"-f", "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml")
	_, err := Run(cmd)
	return err
}

// UninstallGatewayInferenceExtension uninstalls Gateway Inference Extension.
func UninstallGatewayInferenceExtension() error {
	cmd := exec.Command("kubectl", "delete",
		"-f", "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml",
		"--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

// InstallIstioBase installs Istio base.
func InstallIstioBase() error {
	cmd := exec.Command("helm", "repo", "add", "istio", "https://istio-release.storage.googleapis.com/charts")
	if _, err := Run(cmd); err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to add istio helm repo: %w", err)
	}

	cmd = exec.Command("helm", "repo", "update", "istio")
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to update istio helm repo: %w", err)
	}

	cmd = exec.Command("helm", "upgrade", "-i", "istio-base", "istio/base",
		"--version", "1.28.1",
		"-n", "istio-system",
		"--create-namespace")
	_, err := Run(cmd)
	return err
}

// InstallIstiod installs Istiod.
func InstallIstiod(valuesPath string) error {
	helmArgs := []string{
		"upgrade", "-i", "istiod", "istio/istiod",
		"--version", "1.28.1",
		"-n", "istio-system",
	}
	if valuesPath != "" {
		helmArgs = append(helmArgs, "-f", valuesPath)
	}
	cmd := exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

// InstallKgatewayCRDs installs KGateway CRDs.
func InstallKgatewayCRDs() error {
	cmd := exec.Command("helm", "upgrade", "-i", "kgateway-crds",
		"oci://cr.kgateway.dev/kgateway-dev/charts/kgateway-crds",
		"--version", "v2.1.1",
		"-n", "kgateway-system",
		"--create-namespace")
	_, err := Run(cmd)
	return err
}

// InstallKgateway installs KGateway.
func InstallKgateway(valuesPath string) error {
	helmArgs := []string{
		"upgrade", "-i", "kgateway",
		"oci://cr.kgateway.dev/kgateway-dev/charts/kgateway",
		"--version", "v2.1.1",
		"-n", "kgateway-system",
	}
	if valuesPath != "" {
		helmArgs = append(helmArgs, "-f", valuesPath)
	}
	cmd := exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

// UninstallGatewayController uninstalls the gateway controller for the given gateway class.
func UninstallGatewayController(gatewayClass string) error {
	switch gatewayClass {
	case "istio":
		cmd := exec.Command("helm", "uninstall", "istiod", "-n", "istio-system", "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			return err
		}

		cmd = exec.Command("helm", "uninstall", "istio-base", "-n", "istio-system", "--ignore-not-found=true")
		_, err := Run(cmd)
		return err
	case "kgateway":
		cmd := exec.Command("helm", "uninstall", "kgateway", "-n", "kgateway-system", "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			return err
		}

		cmd = exec.Command("helm", "uninstall", "kgateway-crds", "-n", "kgateway-system", "--ignore-not-found=true")
		_, err := Run(cmd)
		return err
	default:
		return fmt.Errorf("unsupported gateway class: %s", gatewayClass)
	}
}
