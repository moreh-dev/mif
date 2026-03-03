//go:build e2e
// +build e2e

package envs

import (
	"fmt"
	"os"
)

const (
	// Skip
	envSkipPrerequisite = "SKIP_PREREQUISITE"
	envSkipCleanup      = "SKIP_CLEANUP"

	// AWS Credentials
	envAWSAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envAWSSecretAccessKey = "AWS_SECRET_ACCESS_KEY"

	// S3 Credentials for performance results
	envS3AccessKeyID     = "S3_ACCESS_KEY_ID"
	envS3SecretAccessKey = "S3_SECRET_ACCESS_KEY"

	// Namespace
	envMIFNamespace      = "MIF_NAMESPACE"
	envWorkloadNamespace = "WORKLOAD_NAMESPACE"

	// Gateway
	envGatewayClassName = "GATEWAY_CLASS_NAME"
	envIstioRev         = "ISTIO_REV"
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
	{envSkipPrerequisite, boolDefaultString(true), "Skip prerequisite installation", "Execution", "bool"},
	{envSkipCleanup, boolDefaultString(false), "Skip cleanup after tests", "Execution", "bool"},

	// AWS Credentials
	{envAWSAccessKeyID, "", "AWS access key ID (for ECR)", "Credentials", "string"},
	{envAWSSecretAccessKey, "", "AWS secret access key (for ECR)", "Credentials", "string"},

	// S3 Credentials for performance results
	{envS3AccessKeyID, "", "AWS access key ID for S3 results upload", "Credentials", "string"},
	{envS3SecretAccessKey, "", "AWS secret access key for S3 results upload", "Credentials", "string"},

	// Namespace
	{envMIFNamespace, "mif", "MIF namespace", "Environment", "string"},
	{envWorkloadNamespace, "mif-e2e-test", "Workload namespace (override for CI isolation)", "Environment", "string"},

	// Gateway
	{envGatewayClassName, "istio", "Gateway class name (istio or kgateway)", "Environment", "string"},
	{envIstioRev, "", "Istio revision label value for workload namespace", "Environment", "optional"},
}
var envVarDefaults = buildEnvVarDefaultMap()

var (
	// Skip
	SkipPrerequisite = getEnvBool(envSkipPrerequisite)
	SkipCleanup      = getEnvBool(envSkipCleanup)

	// AWS Credentials
	AWSAccessKeyID     = getEnv(envAWSAccessKeyID)
	AWSSecretAccessKey = getEnv(envAWSSecretAccessKey)

	// S3 Credentials for performance results
	S3AccessKeyID     = getEnv(envS3AccessKeyID)
	S3SecretAccessKey = getEnv(envS3SecretAccessKey)

	// Namespace
	MIFNamespace      = getEnv(envMIFNamespace)
	WorkloadNamespace = getEnv(envWorkloadNamespace)

	// Gateway
	GatewayClassName = getEnv(envGatewayClassName)
	IstioRev         = getEnv(envIstioRev)
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
