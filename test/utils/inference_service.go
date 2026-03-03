//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

type InferenceServiceData struct {
	Namespace string
}

// CreateInferenceService creates an InferenceService CR in the given namespace.
func CreateInferenceService(namespace string, manifestPath string, data InferenceServiceData) (string, error) {
	rendered, err := renderTemplateFile(manifestPath, data)
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

// GetInferenceServiceContainerImage returns the main container image of the given InferenceService.
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
