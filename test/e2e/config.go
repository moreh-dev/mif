//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"os"
)

// testConfig holds all configuration values for E2E tests.
// Values are initialized from environment variables in the init() function.

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
		skipCertManagerInstall:        getEnvBool(envSkipCertManager, false),
		isCertManagerAlreadyInstalled: false,
		skipCleanup:                   getEnvBool(envSkipCleanup, false),

		testNamespace:   getEnv(envNamespace, "mif"),
		mifChartPath:    getEnv(envMIFChartPath, "deploy/helm/moai-inference-framework"),
		presetChartPath: getEnv(envPresetChartPath, "deploy/helm/moai-inference-preset"),
		testModel:       getEnv(envTestModel, "meta-llama/Llama-3.2-1B-Instruct"),
		gatewayClass:    getEnv(envGatewayClassName, "istio"),

		kindClusterName:               getEnv(envKindClusterName, "mif-e2e"),
		skipKind:                      getEnvBool(envSkipKind, false),
		skipKindDelete:                getEnvBool(envSkipKindDelete, false),
		skipMIFDeploy:                 getEnvBool(envSkipMIFDeploy, false),
		skipPresetDeploy:              getEnvBool(envSkipPresetDeploy, false),
		skipGatewayAPI:                getEnvBool(envSkipGatewayAPI, false),
		skipGatewayInferenceExtension: getEnvBool(envSkipGatewayInferenceExtension, false),
		skipGatewayController:         getEnvBool(envSkipGatewayController, false),
		isUsingKindCluster:            false,

		awsAccessKeyID:     getEnv(envAWSAccessKeyID, ""),
		awsSecretAccessKey: getEnv(envAWSSecretAccessKey, ""),
		hfToken:            getEnv(envHFToken, ""),
		hfEndpoint:         getEnv(envHFEndpoint, ""),

		inferenceImageRepo: getEnv(envInferenceImageRepo, ""),
		inferenceImageTag:  getEnv(envInferenceImageTag, ""),

		kedaEnabled:            getEnvBool(envKEDAEnabled, true),
		lwsEnabled:             getEnvBool(envLWSEnabled, true),
		odinCRDEnabled:         getEnvBool(envOdinCRDEnabled, true),
		prometheusStackEnabled: getEnvBool(envPrometheusStackEnabled, false),
	}
}

// getEnv retrieves an environment variable value or returns the default if not set.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable value.
// Returns true only if the value is exactly "true", otherwise returns the default.
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true"
}
