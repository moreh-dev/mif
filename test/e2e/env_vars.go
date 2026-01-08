//go:build e2e
// +build e2e

package e2e

import "fmt"

type envVarInfo struct {
	Name         string
	Description  string
	DefaultValue string
	Category     string
	Type         string // "string", "bool", "optional"
}

var envVars = []envVarInfo{
	// Skip Installation
	{"SKIP_KIND", "Skip kind cluster creation", "false", "Skip Installation", "bool"},
	{"SKIP_KIND_DELETE", "Skip kind cluster deletion", "false", "Skip Installation", "bool"},
	{"SKIP_CERT_MANAGER", "Skip cert-manager installation", "false", "Skip Installation", "bool"},
	{"SKIP_MIF_DEPLOY", "Skip MIF deployment", "false", "Skip Installation", "bool"},
	{"SKIP_PRESET_DEPLOY", "Skip preset deployment", "false", "Skip Installation", "bool"},
	{"SKIP_GATEWAY_API", "Skip Gateway API installation", "false", "Skip Installation", "bool"},
	{"SKIP_GATEWAY_INFERENCE_EXTENSION", "Skip Gateway Inference Extension installation", "false", "Skip Installation", "bool"},
	{"SKIP_GATEWAY_CONTROLLER", "Skip Gateway controller (Istio/Kgateway) installation", "false", "Skip Installation", "bool"},
	{"SKIP_CLEANUP", "Skip cleanup after tests", "false", "Skip Installation", "bool"},

	// Configuration
	{"NAMESPACE", "Test namespace", "mif", "Configuration", "string"},
	{"MIF_CHART_PATH", "MIF Helm chart path", "deploy/helm/moai-inference-framework", "Configuration", "string"},
	{"PRESET_CHART_PATH", "Preset Helm chart path", "deploy/helm/moai-inference-preset", "Configuration", "string"},
	{"TEST_MODEL", "Test model name", "meta-llama/Llama-3.2-1B-Instruct", "Configuration", "string"},
	{"GATEWAY_CLASS_NAME", "Gateway class (istio or kgateway)", "istio", "Configuration", "string"},
	{"KIND_CLUSTER_NAME", "Kind cluster name", "mif-e2e", "Configuration", "string"},
	{"KIND_K8S_VERSION", "Kubernetes version for kind", "(optional, no default)", "Configuration", "optional"},

	// AWS Credentials
	{"AWS_ACCESS_KEY_ID", "AWS access key ID", "", "AWS Credentials (for ECR)", "string"},
	{"AWS_SECRET_ACCESS_KEY", "AWS secret access key", "", "AWS Credentials (for ECR)", "string"},

	// HuggingFace
	{"HF_TOKEN", "HuggingFace token", "", "HuggingFace", "string"},
	{"HF_ENDPOINT", "HuggingFace endpoint URL", "", "HuggingFace", "string"},

	// Inference Image
	{"INFERENCE_IMAGE_REPO", "Inference image repository", "(optional)", "Inference Image", "optional"},
	{"INFERENCE_IMAGE_TAG", "Inference image tag", "(optional)", "Inference Image", "optional"},

	// Component Enable/Disable
	{"KEDA_ENABLED", "Enable/disable KEDA", "auto-detect", "Component Enable/Disable", "bool"},
	{"LWS_ENABLED", "Enable/disable LWS", "auto-detect", "Component Enable/Disable", "bool"},
	{"ODIN_CRD_ENABLED", "Enable/disable Odin CRD", "auto-detect", "Component Enable/Disable", "bool"},
	{"PROMETHEUS_STACK_ENABLED", "Enable/disable Prometheus Stack", "false", "Component Enable/Disable", "bool"},
}

// getUsedEnvVars returns a map of environment variable names used in init()
// This is used for validation to ensure all used env vars are documented
func getUsedEnvVars() map[string]bool {
	return map[string]bool{
		// From init() function - must match envVars array
		"SKIP_CERT_MANAGER":                true,
		"SKIP_CLEANUP":                     true,
		"NAMESPACE":                        true,
		"MIF_CHART_PATH":                   true,
		"PRESET_CHART_PATH":                true,
		"TEST_MODEL":                       true,
		"GATEWAY_CLASS_NAME":               true,
		"KIND_CLUSTER_NAME":                true,
		"SKIP_KIND":                        true,
		"SKIP_KIND_DELETE":                 true,
		"SKIP_MIF_DEPLOY":                  true,
		"SKIP_PRESET_DEPLOY":               true,
		"SKIP_GATEWAY_API":                 true,
		"SKIP_GATEWAY_INFERENCE_EXTENSION": true,
		"SKIP_GATEWAY_CONTROLLER":          true,
		"AWS_ACCESS_KEY_ID":                true,
		"AWS_SECRET_ACCESS_KEY":            true,
		"HF_TOKEN":                         true,
		"HF_ENDPOINT":                      true,
		"INFERENCE_IMAGE_REPO":             true,
		"INFERENCE_IMAGE_TAG":              true,
		"KEDA_ENABLED":                     true,
		"LWS_ENABLED":                      true,
		"ODIN_CRD_ENABLED":                 true,
		"PROMETHEUS_STACK_ENABLED":         true,
		// Note: KIND_K8S_VERSION is optional and may not be used in init()
		// but is documented for user reference
	}
}

// validateEnvVars ensures all environment variables used in init() are documented in envVars
// This function should be called during tests or build to catch missing documentation
func validateEnvVars() error {
	used := getUsedEnvVars()
	documented := make(map[string]bool)
	for _, env := range envVars {
		documented[env.Name] = true
	}

	var missing []string
	for name := range used {
		if !documented[name] {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("the following environment variables are used in init() but not documented in envVars: %v", missing)
	}

	return nil
}
