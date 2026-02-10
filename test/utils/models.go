//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/moreh-dev/mif/test/utils/settings"
)

// CreateModelPV creates a model PersistentVolume and returns its name.
func CreateModelPV(namespace string) (string, error) {
	data := struct {
		Namespace string
	}{
		Namespace: namespace,
	}

	rendered, err := RenderTemplate(settings.ModelPV, data)
	if err != nil {
		return "", fmt.Errorf("failed to render model PV template: %w", err)
	}
	cmd := exec.Command("kubectl", "apply", "-f", "-", "-o", "name")
	cmd.Stdin = strings.NewReader(rendered)
	output, err := Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create model PV: %w", err)
	}

	pvName := ParseResourceName(output)

	cmd = exec.Command("kubectl", "patch", "pv", pvName, "-p", `{"spec":{"claimRef":null}}`)
	if _, err := Run(cmd); err != nil {
		return "", fmt.Errorf("failed to patch model PV: %w", err)
	}
	return pvName, nil
}

// DeleteModelPV deletes a model PersistentVolume identified by its name.
func DeleteModelPV(pvName string) {
	cmd := exec.Command("kubectl", "delete", "pv", pvName,
		"--ignore-not-found=true")
	_, _ = Run(cmd)
}

// CreateModelPVC creates a model PVC in the given namespace.
func CreateModelPVC(namespace string) (string, error) {
	data := struct {
		Namespace string
	}{
		Namespace: namespace,
	}

	rendered, err := RenderTemplate(settings.ModelPVC, data)
	if err != nil {
		return "", fmt.Errorf("failed to render model PVC template: %w", err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-o", "name")
	cmd.Stdin = strings.NewReader(rendered)
	output, err := Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create model PVC: %w", err)
	}
	return ParseResourceName(output), nil
}

// DeleteModelPVC deletes a model PVC in the given namespace.
func DeleteModelPVC(namespace string, pvcName string) {
	cmd := exec.Command("kubectl", "delete", "pvc", pvcName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = Run(cmd)
}
