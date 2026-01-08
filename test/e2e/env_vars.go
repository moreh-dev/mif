//go:build e2e
// +build e2e

package e2e

import "fmt"

const (
	envSkipKind            = "SKIP_KIND"
	envSkipPrerequisite    = "SKIP_PREREQUISITE"
	envSkipCleanup         = "SKIP_CLEANUP"
	envMIFNamespace        = "MIF_NAMESPACE"
	envWorkloadNamespace   = "WORKLOAD_NAMESPACE"
	envMIFChartPath        = "MIF_CHART_PATH"
	envPresetChartPath     = "PRESET_CHART_PATH"
	envTestModel           = "TEST_MODEL"
	envGatewayClassName    = "GATEWAY_CLASS_NAME"
	envKindClusterName     = "KIND_CLUSTER_NAME"
	envKindK8sVersion      = "KIND_K8S_VERSION"
	envAWSAccessKeyID      = "AWS_ACCESS_KEY_ID"
	envAWSSecretAccessKey  = "AWS_SECRET_ACCESS_KEY"
	envHFToken             = "HF_TOKEN"
	envHFEndpoint          = "HF_ENDPOINT"
	envInferenceImageRepo  = "INFERENCE_IMAGE_REPO"
	envInferenceImageTag   = "INFERENCE_IMAGE_TAG"
	envKEDAEnabled         = "KEDA_ENABLED"
	envLWSEnabled          = "LWS_ENABLED"
	envOdinCRDEnabled      = "ODIN_CRD_ENABLED"
	envPrometheusStackEnabled = "PROMETHEUS_STACK_ENABLED"
)

type envVarInfo struct {
	Name         string
	Description  string
	DefaultValue string
	Category     string
	Type         string // "string", "bool", "optional"
}

var envVars = []envVarInfo{
	// Skip
	{envSkipKind, "Skip kind cluster creation and deletion", "false", "Skip", "bool"},
	{envSkipPrerequisite, "Skip prerequisite installation (cert-manager, Gateway API, Gateway controller, Gateway Inference Extension)", "false", "Skip", "bool"},
	{envSkipCleanup, "Skip cleanup after tests", "false", "Skip", "bool"},

	// Configuration
	{envMIFNamespace, "MIF namespace", "mif", "Configuration", "string"},
	{envWorkloadNamespace, "Workload namespace for InferenceService", "quickstart", "Configuration", "string"},
	{envMIFChartPath, "MIF Helm chart path", "deploy/helm/moai-inference-framework", "Configuration", "string"},
	{envPresetChartPath, "Preset Helm chart path", "deploy/helm/moai-inference-preset", "Configuration", "string"},
	{envTestModel, "Test model name", "meta-llama/Llama-3.2-1B-Instruct", "Configuration", "string"},
	{envGatewayClassName, "Gateway class (istio or kgateway)", "istio", "Configuration", "string"},
	{envKindClusterName, "Kind cluster name", "mif-e2e", "Configuration", "string"},
	{envKindK8sVersion, "Kubernetes version for kind", "(optional, no default)", "Configuration", "optional"},

	// AWS Credentials
	{envAWSAccessKeyID, "AWS access key ID", "", "AWS Credentials (for ECR)", "string"},
	{envAWSSecretAccessKey, "AWS secret access key", "", "AWS Credentials (for ECR)", "string"},

	// HuggingFace
	{envHFToken, "HuggingFace token", "", "HuggingFace", "string"},
	{envHFEndpoint, "HuggingFace endpoint URL", "", "HuggingFace", "string"},

	// Inference Image
	{envInferenceImageRepo, "Inference image repository", "(optional)", "Inference Image", "optional"},
	{envInferenceImageTag, "Inference image tag", "(optional)", "Inference Image", "optional"},

	// Component Enable/Disable
	{envKEDAEnabled, "Enable/disable KEDA", "auto-detect", "Component Enable/Disable", "bool"},
	{envLWSEnabled, "Enable/disable LWS", "auto-detect", "Component Enable/Disable", "bool"},
	{envOdinCRDEnabled, "Enable/disable Odin CRD", "auto-detect", "Component Enable/Disable", "bool"},
	{envPrometheusStackEnabled, "Enable/disable Prometheus Stack", "false", "Component Enable/Disable", "bool"},
}

// getUsedEnvVars returns environment variable names used in config.go init().
func getUsedEnvVars() map[string]bool {
	used := make(map[string]bool)
	for _, env := range envVars {
		if env.Name != envKindK8sVersion {
			used[env.Name] = true
		}
	}
	return used
}

// validateEnvVars ensures all environment variables used in config.go init() are documented in envVars.
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
		return fmt.Errorf("the following environment variables are used in config.go init() but not documented in envVars: %v", missing)
	}

	return nil
}