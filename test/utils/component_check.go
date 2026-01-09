package utils

import (
	"os/exec"
	"strings"
)

// IsKEDAInstalled checks if KEDA is installed.
func IsKEDAInstalled() bool {
	kedaCRDs := []string{
		"scaledobjects.keda.sh",
		"scaledjobs.keda.sh",
		"cloudeventsources.eventing.keda.sh",
		"clustertriggerauthentications.keda.sh",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	foundCRDs := make(map[string]bool, len(kedaCRDs))
	for _, crd := range kedaCRDs {
		foundCRDs[crd] = false
	}

	for _, line := range crdList {
		for _, crd := range kedaCRDs {
			if strings.Contains(line, crd) {
				foundCRDs[crd] = true
				break
			}
		}
	}

	allCRDsPresent := true
	for _, found := range foundCRDs {
		if !found {
			allCRDsPresent = false
			break
		}
	}

	if allCRDsPresent {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^keda$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

// IsLWSInstalled checks if LWS is installed.
func IsLWSInstalled() bool {
	lwsCRDs := []string{
		"leaderworkersets.leaderworkerset.x-k8s.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	allCRDsPresent := true
	for _, crd := range lwsCRDs {
		found := false
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				found = true
				break
			}
		}
		if !found {
			allCRDsPresent = false
			break
		}
	}

	if allCRDsPresent {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^lws$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

// IsOdinCRDInstalled checks if Odin CRDs are installed.
func IsOdinCRDInstalled() bool {
	odinCRDs := []string{
		"inferenceservices.odin.moreh.io",
		"inferenceservicetemplates.odin.moreh.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	crdFound := make(map[string]bool, len(odinCRDs))
	for _, line := range crdList {
		for _, crd := range odinCRDs {
			if strings.Contains(line, crd) {
				crdFound[crd] = true
			}
		}
	}

	allFound := true
	for _, crd := range odinCRDs {
		if !crdFound[crd] {
			allFound = false
			break
		}
	}

	if allFound {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^odin-crd$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

// IsPrometheusInstalled checks if Prometheus Stack is installed.
func IsPrometheusInstalled() bool {
	prometheusCRDs := []string{
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"servicemonitors.monitoring.coreos.com",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	foundCRDs := make(map[string]bool, len(prometheusCRDs))
	for _, crd := range prometheusCRDs {
		foundCRDs[crd] = false
	}

	for _, line := range crdList {
		for crd := range foundCRDs {
			if strings.Contains(line, crd) {
				foundCRDs[crd] = true
			}
		}
	}

	allPresent := true
	for _, present := range foundCRDs {
		if !present {
			allPresent = false
			break
		}
	}

	if allPresent {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^kube-prometheus-stack$|^prometheus-stack$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	cmd = exec.Command("kubectl", "get", "prometheus", "-A", "--no-headers")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

// IsMIFInstalled checks if moai-inference-framework is installed in the given namespace.
func IsMIFInstalled(namespace string) bool {
	cmd := exec.Command("helm", "list", "-n", namespace, "-q", "-f", "^moai-inference-framework$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}

// IsPresetInstalled checks if moai-inference-preset is installed in the given namespace.
func IsPresetInstalled(namespace string) bool {
	cmd := exec.Command("helm", "list", "-n", namespace, "-q", "-f", "^moai-inference-preset$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}

// IsGatewayAPIInstalled checks if Gateway API is installed.
func IsGatewayAPIInstalled() bool {
	cmd := exec.Command("kubectl", "get", "crd", "gateways.gateway.networking.k8s.io", "--ignore-not-found")
	output, err := Run(cmd)
	if err == nil && strings.Contains(output, "gateways.gateway.networking.k8s.io") {
		return true
	}
	return false
}

// IsGatewayInferenceExtensionInstalled checks if Gateway Inference Extension is installed.
func IsGatewayInferenceExtensionInstalled() bool {
	cmd := exec.Command("kubectl", "get", "crd", "inferencepools.inference.networking.k8s.io", "--ignore-not-found")
	output, err := Run(cmd)
	if err == nil && strings.Contains(output, "inferencepools.inference.networking.k8s.io") {
		return true
	}
	return false
}

// IsIstioInstalled checks if Istio is installed.
func IsIstioInstalled() bool {
	cmd := exec.Command("helm", "list", "-n", "istio-system", "-q", "-f", "^istiod$|^istio-base$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}

// IsKgatewayInstalled checks if KGateway is installed.
func IsKgatewayInstalled() bool {
	cmd := exec.Command("helm", "list", "-n", "kgateway-system", "-q", "-f", "^kgateway$|^kgateway-crds$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}
