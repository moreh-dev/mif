//go:build e2e
// +build e2e

package envs

import (
	"fmt"
	"os"
)

const (
	// Skip
	envSkipKind         = "SKIP_KIND"
	envSkipPrerequisite = "SKIP_PREREQUISITE"
	envSkipCleanup      = "SKIP_CLEANUP"

	// Test Model
	envTestModel = "TEST_MODEL"

	// AWS Credentials
	envAWSAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envAWSSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	envS3AccessKeyID      = "S3_ACCESS_KEY_ID"
	envS3SecretAccessKey  = "S3_SECRET_ACCESS_KEY"
	envS3Region           = "S3_REGION"
	envS3Bucket           = "S3_BUCKET"

	// HuggingFace
	envHFToken    = "HF_TOKEN"
	envHFEndpoint = "HF_ENDPOINT"

	// Benchmark
	envQualityBenchmarks     = "QUALITY_BENCHMARKS"
	envQualityBenchmarkLimit = "QUALITY_BENCHMARK_LIMIT"

	// Namespace
	envMIFNamespace      = "MIF_NAMESPACE"
	envWorkloadNamespace = "WORKLOAD_NAMESPACE"

	// Gateway Class
	envGatewayClassName = "GATEWAY_CLASS_NAME"

	// Istio
	envIstioRev = "ISTIO_REV"
)

type envVarInfo struct {
	Name         string
	DefaultValue string
	Description  string
	Category     string
	Type         string // "string", "bool", "optional"
}

var envVars = []envVarInfo{
	// Skip
	{envSkipKind, boolDefaultString(false), "Skip kind cluster creation and deletion", "Skip", "bool"},
	{envSkipPrerequisite, boolDefaultString(false), "Skip prerequisite installation (cert-manager, Gateway API, Gateway controller, Gateway Inference Extension) and MIF/Preset setup. When enabled, setupPrerequisites() returns early without installing or validating any components", "Skip", "bool"},
	{envSkipCleanup, boolDefaultString(false), "Skip cleanup after tests", "Skip", "bool"},

	// Test Model
	{envTestModel, "Qwen/Qwen3-0.6B", "Test model name", "Configuration", "string"},

	// AWS Credentials
	{envAWSAccessKeyID, "", "AWS access key ID", "AWS Credentials (for ECR)", "string"},
	{envAWSSecretAccessKey, "", "AWS secret access key", "AWS Credentials (for ECR)", "string"},

	// S3 Credentials
	{envS3AccessKeyID, "", "AWS access key ID for S3 results upload", "AWS Credentials (for S3)", "string"},
	{envS3SecretAccessKey, "", "AWS secret access key for S3 results upload", "AWS Credentials (for S3)", "string"},
	{envS3Region, "ap-northeast-2", "AWS region for S3 results bucket", "AWS Credentials (for S3)", "string"},
	{envS3Bucket, "moreh-benchmark", "S3 bucket name for inference-perf results", "AWS Credentials (for S3)", "string"},

	// HuggingFace
	{envHFToken, "", "HuggingFace token", "HuggingFace", "string"},
	{envHFEndpoint, "", "HuggingFace endpoint URL", "HuggingFace", "string"},

	// Benchmark
	{envQualityBenchmarks, "mmlu", "Name of a single quality benchmark to run (for example: sample, mmlu, gsm8k_cot, hellaswag, aime, gpqa; this list is not exhaustive)", "Benchmark", "string"},
	{envQualityBenchmarkLimit, "", "Optional limit hint for benchmark dataset; actual usage is defined by the benchmark implementation", "Benchmark", "string"},

	// Namespace
	{envMIFNamespace, "mif", "MIF namespace", "Namespace", "string"},
	{envWorkloadNamespace, "mif-e2e-test", "Workload namespace", "Namespace", "string"},

	// Gateway Class
	{envGatewayClassName, "istio", "Gateway class (istio or kgateway)", "Gateway Class", "string"},

	// Istio
	{envIstioRev, "", "Istio revision label value for workload namespace", "Istio", "optional"},
}
var envVarDefaults = buildEnvVarDefaultMap()

var (
	// Skip
	SkipKind         = getEnvBool(envSkipKind)
	SkipPrerequisite = getEnvBool(envSkipPrerequisite)
	SkipCleanup      = getEnvBool(envSkipCleanup)

	// Test Model
	TestModel = getEnv(envTestModel)

	// AWS Credentials
	AWSAccessKeyID     = getEnv(envAWSAccessKeyID)
	AWSSecretAccessKey = getEnv(envAWSSecretAccessKey)

	// S3 Credentials
	S3AccessKeyID     = getEnv(envS3AccessKeyID)
	S3SecretAccessKey = getEnv(envS3SecretAccessKey)
	S3Region          = getEnv(envS3Region)
	S3Bucket          = getEnv(envS3Bucket)

	// HuggingFace
	HFToken    = getEnv(envHFToken)
	HFEndpoint = getEnv(envHFEndpoint)

	// Benchmark
	QualityBenchmarks     = getEnv(envQualityBenchmarks)
	QualityBenchmarkLimit = getEnv(envQualityBenchmarkLimit)

	// Namespace
	MIFNamespace      = getEnv(envMIFNamespace)
	WorkloadNamespace = getEnv(envWorkloadNamespace)

	// Gateway Class
	GatewayClassName = getEnv(envGatewayClassName)

	// Istio
	IstioRev = getEnv(envIstioRev)
)

func boolDefaultString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func buildEnvVarDefaultMap() map[string]string {
	m := make(map[string]string, len(envVars))
	for _, e := range envVars {
		m[e.Name] = e.DefaultValue
	}
	return m
}

// getUsedEnvVars returns environment variable names used in config.go init().
func getUsedEnvVars() map[string]bool {
	used := make(map[string]bool)
	for _, env := range envVars {
		used[env.Name] = true
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

func getEnv(key string) string {
	defaultValue := envVarDefaults[key]
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) bool {
	defaultValue := envVarDefaults[key] == "true"
	if value := os.Getenv(key); value != "" {
		return value == "true"
	}
	return defaultValue
}
