//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/moreh-dev/mif/test/utils/settings"
)

type InferenceServiceData struct {
	Name            string
	Namespace       string
	ImagePullSecret string
	Image           string
	Model           string
	HFToken         string
	HFEndpoint      string
	IsKind          bool
}

func GetInferenceImageInfo(isKind bool) (repo, tag string) {
	repo = settings.ImageRepoDefault
	tag = settings.ImageTagDefault
	if isKind {
		repo = settings.ImageRepoKindDefault
		tag = settings.ImageTagKindDefault
	}

	return repo, tag
}

func GetInferenceServiceData(namespace string, model string, hfToken string, hfEndpoint string, isKind bool) InferenceServiceData {
	imageRepo, imageTag := GetInferenceImageInfo(isKind)
	image := fmt.Sprintf("%s:%s", imageRepo, imageTag)

	return InferenceServiceData{
		Name:            settings.InferenceServiceName,
		Namespace:       namespace,
		ImagePullSecret: settings.MorehRegistrySecretName,
		Image:           image,
		Model:           model,
		HFToken:         hfToken,
		HFEndpoint:      hfEndpoint,
		IsKind:          isKind,
	}
}

// CreateInferenceService creates an InferenceService CR in the given namespace.
func CreateInferenceService(namespace string, manifestPath string, data InferenceServiceData) error {
	rendered, err := RenderTemplate(manifestPath, data)
	if err != nil {
		return fmt.Errorf("failed to render InferenceService manifest: %w", err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace)
	cmd.Stdin = strings.NewReader(rendered)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to create InferenceService: %w", err)
	}
	return nil
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
func CreateInferenceServiceTemplate(namespace string, manifestPath string, data InferenceServiceData) error {
	rendered, err := RenderTemplate(manifestPath, data)
	if err != nil {
		return fmt.Errorf("failed to render InferenceServiceTemplate manifest: %w", err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace)
	cmd.Stdin = strings.NewReader(rendered)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to create InferenceServiceTemplate: %w", err)
	}
	return nil
}

// DeleteInferenceServiceTemplate deletes an InferenceServiceTemplate from the given namespace.
func DeleteInferenceServiceTemplate(namespace, templateName string) {
	cmd := exec.Command("kubectl", "delete", "inferenceservicetemplate", templateName,
		"-n", namespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
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
