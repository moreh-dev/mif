//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

type InferenceServiceData struct {
	Name         string
	Namespace    string
	TemplateRefs []string
	HFToken      string
	HFEndpoint   string
}

func GetInferenceServiceData(name string, namespace string, templateRefs []string, hfToken string, hfEndpoint string) InferenceServiceData {
	return InferenceServiceData{
		Name:         name,
		Namespace:    namespace,
		TemplateRefs: templateRefs,
		HFToken:      hfToken,
		HFEndpoint:   hfEndpoint,
	}
}

// CreateInferenceService creates an InferenceService CR in the given namespace.
func CreateInferenceService(namespace string, manifestPath string, data InferenceServiceData) (string, error) {
	rendered, err := RenderTemplate(manifestPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to render InferenceService manifest: %w", err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace, "-o", "name")
	cmd.Stdin = strings.NewReader(rendered)
	output, err := Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create InferenceService: %w", err)
	}
	return ParseResourceName(output), nil
}

// DeleteInferenceService deletes an InferenceService from the given namespace.
func DeleteInferenceService(namespace string, name string) {
	cmd := exec.Command("kubectl", "delete", "inferenceservice", name,
		"-n", namespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// CreateInferenceServiceTemplate creates an InferenceServiceTemplate CR in the given namespace.
func CreateInferenceServiceTemplate(namespace string, manifestPath string) (string, error) {
	rendered, err := RenderTemplate(manifestPath, map[string]string{"Namespace": namespace})
	if err != nil {
		return "", fmt.Errorf("failed to render InferenceServiceTemplate manifest: %w", err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace, "-o", "name")
	cmd.Stdin = strings.NewReader(rendered)
	output, err := Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create InferenceServiceTemplate: %w", err)
	}
	return ParseResourceName(output), nil
}

// DeleteInferenceServiceTemplate deletes an InferenceServiceTemplate from the given namespace.
func DeleteInferenceServiceTemplate(namespace, templateName string) {
	cmd := exec.Command("kubectl", "delete", "inferenceservicetemplate", templateName,
		"-n", namespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// GetInferenceServiceContainerImage returns the main container image of the given InferenceService.
// Call after the service is created and merged (e.g. when Ready); uses the actual applied spec.
func GetInferenceServiceContainerImage(namespace, serviceName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "deployment", serviceName,
		"-n", namespace,
		"-o", "jsonpath={.spec.template.spec.containers[0].image}")
	output, err := Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get Deployment container image: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// GetGatewayServiceName gets the name of the Gateway service in the workload namespace.
func GetGatewayServiceName(namespace string) (string, error) {
	cmd := exec.Command("kubectl", "get", "service",
		"-n", namespace,
		"-l", "gateway.networking.k8s.io/gateway-name=mif",
		"-o", "jsonpath={.items[0].metadata.name}")

	output, err := Run(cmd)
	if err != nil {
		return "", fmt.Errorf("gateway service not found: %w", err)
	}
	return strings.TrimSpace(output), nil
}
