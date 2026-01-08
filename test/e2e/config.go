//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"os"
)

type testConfig struct {
	skipCertManagerInstall        bool
	isCertManagerAlreadyInstalled bool
	skipCleanup                   bool

	testNamespace   string
	mifChartPath    string
	presetChartPath string
	testModel       string
	gatewayClass    string

	kindClusterName               string
	skipKind                      bool
	skipKindDelete                bool
	skipMIFDeploy                 bool
	skipPresetDeploy              bool
	skipGatewayAPI                bool
	skipGatewayInferenceExtension bool
	skipGatewayController         bool
	isUsingKindCluster            bool

	isMIFAlreadyInstalled                bool
	isPresetAlreadyInstalled             bool
	isGatewayAPIAlreadyInstalled         bool
	isGatewayInferenceExtensionInstalled bool
	isIstioAlreadyInstalled              bool
	isKgatewayAlreadyInstalled           bool

	awsAccessKeyID     string
	awsSecretAccessKey string
	hfToken            string
	hfEndpoint         string

	inferenceImageRepo string
	inferenceImageTag  string

	kedaEnabled            bool
	lwsEnabled             bool
	odinCRDEnabled         bool
	prometheusStackEnabled bool
}

var cfg testConfig

func init() {
	cfg = testConfig{
		skipCertManagerInstall:        getEnvBool("SKIP_CERT_MANAGER", false),
		isCertManagerAlreadyInstalled: false,
		skipCleanup:                   getEnvBool("SKIP_CLEANUP", false),

		testNamespace:   getEnv("NAMESPACE", "mif"),
		mifChartPath:    getEnv("MIF_CHART_PATH", "deploy/helm/moai-inference-framework"),
		presetChartPath: getEnv("PRESET_CHART_PATH", "deploy/helm/moai-inference-preset"),
		testModel:       getEnv("TEST_MODEL", "meta-llama/Llama-3.2-1B-Instruct"),
		gatewayClass:    getEnv("GATEWAY_CLASS_NAME", "istio"),

		kindClusterName:               getEnv("KIND_CLUSTER_NAME", "mif-e2e"),
		skipKind:                      getEnvBool("SKIP_KIND", false),
		skipKindDelete:                getEnvBool("SKIP_KIND_DELETE", false),
		skipMIFDeploy:                 getEnvBool("SKIP_MIF_DEPLOY", false),
		skipPresetDeploy:              getEnvBool("SKIP_PRESET_DEPLOY", false),
		skipGatewayAPI:                getEnvBool("SKIP_GATEWAY_API", false),
		skipGatewayInferenceExtension: getEnvBool("SKIP_GATEWAY_INFERENCE_EXTENSION", false),
		skipGatewayController:         getEnvBool("SKIP_GATEWAY_CONTROLLER", false),
		isUsingKindCluster:            false,

		awsAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		awsSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		hfToken:            getEnv("HF_TOKEN", ""),
		hfEndpoint:         getEnv("HF_ENDPOINT", ""),

		inferenceImageRepo: getEnv("INFERENCE_IMAGE_REPO", ""),
		inferenceImageTag:  getEnv("INFERENCE_IMAGE_TAG", ""),

		kedaEnabled:            getEnvBool("KEDA_ENABLED", true),
		lwsEnabled:             getEnvBool("LWS_ENABLED", true),
		odinCRDEnabled:         getEnvBool("ODIN_CRD_ENABLED", true),
		prometheusStackEnabled: getEnvBool("PROMETHEUS_STACK_ENABLED", false),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true"
}
