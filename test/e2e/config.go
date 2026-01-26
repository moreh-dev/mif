//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"os"
)

type testConfig struct {
	skipPrerequisite                     bool
	isCertManagerAlreadyInstalled        bool
	isGatewayAPIAlreadyInstalled         bool
	isGatewayInferenceExtensionInstalled bool
	isIstioAlreadyInstalled              bool
	isKgatewayAlreadyInstalled           bool
	skipCleanup                          bool

	mifNamespace      string
	workloadNamespace string
	mifChartPath      string
	presetChartPath   string
	testModel         string
	gatewayClass      string

	kindClusterName    string
	skipKind           bool
	isUsingKindCluster bool

	isMIFAlreadyInstalled    bool
	isPresetAlreadyInstalled bool

	awsAccessKeyID     string
	awsSecretAccessKey string
	s3AccessKeyID      string
	s3SecretAccessKey  string
	s3Region           string
	s3Bucket           string
	hfToken            string
	hfEndpoint         string

	inferenceImageRepo string
	inferenceImageTag  string
	istioRev           string

	kedaEnabled            bool
	lwsEnabled             bool
	odinCRDEnabled         bool
	prometheusStackEnabled bool

	inferencePerfEnabled    bool
	qualityBenchmarkEnabled bool
	qualityBenchmarks       string
	qualityBenchmarkLimit   string
}

var cfg testConfig

func init() {
	cfg = testConfig{
		skipPrerequisite:                     getEnvBool(envSkipPrerequisite, false),
		isCertManagerAlreadyInstalled:        false,
		isGatewayAPIAlreadyInstalled:         false,
		isGatewayInferenceExtensionInstalled: false,
		isIstioAlreadyInstalled:              false,
		isKgatewayAlreadyInstalled:           false,
		skipCleanup:                          getEnvBool(envSkipCleanup, false),

		mifNamespace:      getEnv(envMIFNamespace, "mif"),
		workloadNamespace: getEnv(envWorkloadNamespace, "quickstart"),
		mifChartPath:      getEnv(envMIFChartPath, "deploy/helm/moai-inference-framework"),
		presetChartPath:   getEnv(envPresetChartPath, "deploy/helm/moai-inference-preset"),
		testModel:         getEnv(envTestModel, "Qwen/Qwen3-0.6B"),
		gatewayClass:      getEnv(envGatewayClassName, "istio"),

		kindClusterName:    getEnv(envKindClusterName, "mif-e2e"),
		skipKind:           getEnvBool(envSkipKind, false),
		isUsingKindCluster: false,

		isMIFAlreadyInstalled:    false,
		isPresetAlreadyInstalled: false,

		awsAccessKeyID:     getEnv(envAWSAccessKeyID, ""),
		awsSecretAccessKey: getEnv(envAWSSecretAccessKey, ""),
		s3AccessKeyID:      getEnv(envS3AccessKeyID, ""),
		s3SecretAccessKey:  getEnv(envS3SecretAccessKey, ""),
		s3Region:           getEnv(envS3Region, "ap-northeast-2"),
		s3Bucket:           getEnv(envS3Bucket, "moreh-benchmark"),
		hfToken:            getEnv(envHFToken, ""),
		hfEndpoint:         getEnv(envHFEndpoint, ""),

		inferenceImageRepo: getEnv(envInferenceImageRepo, ""),
		inferenceImageTag:  getEnv(envInferenceImageTag, ""),
		istioRev:           getEnv(envIstioRev, ""),

		kedaEnabled:            getEnvBool(envKEDAEnabled, true),
		lwsEnabled:             getEnvBool(envLWSEnabled, true),
		odinCRDEnabled:         getEnvBool(envOdinCRDEnabled, true),
		prometheusStackEnabled: getEnvBool(envPrometheusStackEnabled, false),

		inferencePerfEnabled:    getEnvBool(envInferencePerfEnabled, false),
		qualityBenchmarkEnabled: getEnvBool(envQualityBenchmarkEnabled, false),
		qualityBenchmarks:       getEnv(envQualityBenchmarks, "mmlu"),
		qualityBenchmarkLimit:   getEnv(envQualityBenchmarkLimit, ""),
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
