//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moreh-dev/mif/test/utils"
)

// writeValuesFile is a helper function to write Helm values files.
func writeValuesFile(relativePath, content string, mode os.FileMode) (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	valuesPath := filepath.Join(projectDir, relativePath)
	if err := os.WriteFile(valuesPath, []byte(content), mode); err != nil {
		return "", fmt.Errorf("failed to write values file %s: %w", relativePath, err)
	}

	return valuesPath, nil
}

// createMIFValuesFile creates a values file for moai-inference-framework with ECR token refresher configuration.
func createMIFValuesFile(awsAccessKeyID, awsSecretAccessKey string) (string, error) {
	valuesContent := fmt.Sprintf(`ecrTokenRefresher:
  aws:
    accessKeyId: %s
    secretAccessKey: %s
`, awsAccessKeyID, awsSecretAccessKey)

	return writeValuesFile(tempFileMIFValues, valuesContent, 0600)
}

// createHeimdallValuesFile creates a values file for Heimdall.
func createHeimdallValuesFile() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	serviceMonitorSection := "serviceMonitor:\n  enabled: false\n"
	if cfg.prometheusStackEnabled {
		serviceMonitorSection = `serviceMonitor:
  enabled: true
  labels:
    release: prometheus-stack
`
	}

	baseYAML := fmt.Sprintf(`global:
  imagePullSecrets:
    - name: %s

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: single-profile-handler
    - type: queue-scorer
    - type: max-score-picker
  schedulingProfiles:
    - name: default
      plugins:
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker

gateway:
  name: %s
  gatewayClassName: %s

%s`, secretNameMorehRegistry, gatewayName, cfg.gatewayClass, serviceMonitorSection)

	valuesPath := filepath.Join(projectDir, tempFileHeimdallValues)
	err = os.WriteFile(valuesPath, []byte(baseYAML), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write Heimdall values file: %w", err)
	}

	return valuesPath, nil
}
