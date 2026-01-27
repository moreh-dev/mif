//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/moreh-dev/mif/test/utils/settings"
)

// InstallGatewayAPI installs Gateway API.
func InstallGatewayAPI() error {
	cmd := exec.Command("kubectl", "apply", "--server-side",
		"-f", settings.GatewayAPIYAML)
	_, err := Run(cmd)
	return err
}

// UninstallGatewayAPI uninstalls Gateway API.
func UninstallGatewayAPI() {
	cmd := exec.Command("kubectl", "delete",
		"-f", settings.GatewayAPIYAML,
		"--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallGatewayInferenceExtension installs Gateway Inference Extension.
func InstallGatewayInferenceExtension() error {
	cmd := exec.Command("kubectl", "apply",
		"-f", settings.GatewayAPIInferenceExtensionYAML)
	_, err := Run(cmd)
	return err
}

// UninstallGatewayInferenceExtension uninstalls Gateway Inference Extension.
func UninstallGatewayInferenceExtension() {
	cmd := exec.Command("kubectl", "delete",
		"-f", settings.GatewayAPIInferenceExtensionYAML,
		"--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// IsGatewayAPICRDsInstalled checks if any Gateway API CRDs are installed
// by verifying the existence of key CRDs related to Gateway API.
func IsGatewayAPICRDsInstalled() bool {
	// List of common Gateway API CRDs
	gatewayAPICRDs := []string{
		"gatewayclasses.gateway.networking.k8s.io",
		"gateways.gateway.networking.k8s.io",
		"grpcroutes.gateway.networking.k8s.io",
		"httproutes.gateway.networking.k8s.io",
		"inferencepools.inference.networking.k8s.io",
		"referencegrants.gateway.networking.k8s.io",
	}

	// Execute the kubectl command to get all CRDs
	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	// Check if any of the Cert Manager CRDs are present
	crdList := GetNonEmptyLines(output)
	for _, crd := range gatewayAPICRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// InstallGatewayController installs the gateway controller for the given gateway class.
func InstallGatewayController(gatewayClass string) error {
	switch gatewayClass {
	case "istio":
		cmd := exec.Command("helm", "repo", "add", "istio", settings.IstioHelmRepoURL)
		if _, err := Run(cmd); err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to add istio helm repo: %w", err)
		}

		cmd = exec.Command("helm", "repo", "update", "istio")
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to update istio helm repo: %w", err)
		}

		cmd = exec.Command("helm", "upgrade", "-i", "istio-base", "istio/base",
			"--version", settings.IstioVersion,
			"-n", settings.IstioNamespace,
			"--create-namespace")
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to install istio-base: %w", err)
		}

		cmd = exec.Command("helm", "upgrade", "-i", "istiod", "istio/istiod",
			"--version", settings.IstioVersion,
			"-n", settings.IstioNamespace,
			"-f", settings.IstiodValuesFile)
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to install istiod: %w", err)
		}

	case "kgateway":
		cmd := exec.Command("helm", "upgrade", "-i", "kgateway-crds",
			settings.KgatewayCrdsHelmRepoURL,
			"--version", settings.KgatewayCrdsVersion,
			"-n", settings.KgatewayNamespace,
			"--create-namespace")
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to install kgateway crds: %w", err)
		}

		cmd = exec.Command("helm", "upgrade", "-i", "kgateway",
			settings.KgatewayHelmRepoURL,
			"--version", settings.KgatewayVersion,
			"-n", settings.KgatewayNamespace,
			"-f", settings.KgatewayValuesFile)
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to install kgateway: %w", err)
		}

	default:
		return fmt.Errorf("unsupported gateway class: %s", gatewayClass)
	}

	return nil
}

// UninstallGatewayController uninstalls the gateway controller for the given gateway class.
func UninstallGatewayController(gatewayClass string) {
	switch gatewayClass {
	case "istio":
		cmd := exec.Command("helm", "uninstall", "istiod", "-n", settings.IstioNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}

		cmd = exec.Command("helm", "uninstall", "istio-base", "-n", settings.IstioNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}
	case "kgateway":
		cmd := exec.Command("helm", "uninstall", "kgateway", "-n", settings.KgatewayNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}

		cmd = exec.Command("helm", "uninstall", "kgateway-crds", "-n", settings.KgatewayNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}
	default:
		warnError(fmt.Errorf("unsupported gateway class: %s", gatewayClass))
	}
}

// CreateGatewayResource creates Gateway resources in the workload namespace.
func CreateGatewayResource(namespace string, gatewayClass string) error {
	var templatePath string
	switch gatewayClass {
	case "istio":
		templatePath = settings.IstioGatewayTemplate
	case "kgateway":
		templatePath = settings.KgatewayGatewayTemplate
	default:
		return fmt.Errorf("unsupported gateway class: %s", gatewayClass)
	}

	data := struct {
		GatewayName string
	}{
		GatewayName: settings.GatewayName,
	}
	rendered, err := RenderTemplate(templatePath, data)
	if err != nil {
		return fmt.Errorf("failed to render gateway template: %w", err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace, "--request-timeout=60s")
	cmd.Stdin = strings.NewReader(rendered)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to apply Gateway resources: %w", err)
	}

	cmd = exec.Command("kubectl", "wait", "gateway", settings.GatewayName,
		"--for=condition=Programmed",
		"-n", namespace,
		fmt.Sprintf("--timeout=%v", settings.TimeoutLong))
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to wait for Gateway to be programmed: %w", err)
	}
	return nil
}

// DeleteGatewayResource deletes Gateway resources from the given namespace.
func DeleteGatewayResource(namespace, gatewayClass string) {
	cmd := exec.Command("kubectl", "delete", "gateway", settings.GatewayName,
		"-n", namespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(fmt.Errorf("failed to delete Gateway: %w", err))
	}

	switch gatewayClass {
	case "istio":
		cmd := exec.Command("kubectl", "delete", "configmap", "mif-gateway-infrastructure",
			"-n", namespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(fmt.Errorf("failed to delete Gateway ConfigMap: %w", err))
		}
	case "kgateway":
		cmd := exec.Command("kubectl", "delete", "gatewayparameters", "mif-gateway-infrastructure",
			"-n", namespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(fmt.Errorf("failed to delete Gateway GatewayParameters: %w", err))
		}
	}
}
