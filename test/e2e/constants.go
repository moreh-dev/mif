//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import "time"

// Resource names used in E2E tests
const (
	// Helm release names
	helmReleaseMIF      = "moai-inference-framework"
	helmReleaseHeimdall = "heimdall"

	// Gateway class types
	gatewayClassIstio    = "istio"
	gatewayClassKgateway = "kgateway"

	// Test resource names
	testInferenceServiceName = "pd-disaggregation-test"
	gatewayName              = "mif"

	// Kubernetes resource names
	secretNameMorehRegistry = "moreh-registry"
	secretNameECRCreds      = "moai-inference-framework-ecr-token-refresher"
	cronJobNameECRRefresher = "moai-inference-framework-ecr-token-refresher"
	jobNameECRRefresher     = "ecr-token-refresher-init-manual"

	// Helm repository
	helmRepoName = "moreh"
	helmRepoURL  = "https://moreh-dev.github.io/helm-charts"

	// Image repositories (for kind cluster default)
	imageRepoKindDefault = "ghcr.io/llm-d/llm-d-inference-sim"

	// Image tags (for kind cluster default)
	imageTagKindDefault = "v0.6.1"
)

// Timeout and interval constants for E2E tests
const (
	timeoutShort    = 2 * time.Minute
	timeoutMedium   = 5 * time.Minute
	timeoutLong     = 10 * time.Minute
	timeoutVeryLong = 15 * time.Minute

	intervalShort  = 2 * time.Second
	intervalMedium = 5 * time.Second
	intervalLong   = 10 * time.Second
)

// File paths for temporary test files
const (
	tempFileMIFValues      = "test/e2e/moai-inference-framework-values.yaml"
	tempFileHeimdallValues = "test/e2e/heimdall-values.yaml"
	tempFileISValues       = "test/e2e/inference-service-values.yaml"
	tempFileIstiodValues   = "test/e2e/istiod-values.yaml"
	tempFileKgatewayValues = "test/e2e/kgateway-values.yaml"
)
