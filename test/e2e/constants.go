//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import "time"

// Resource names used in E2E tests
const (
	// Helm release names
	helmReleaseMIF      = "moai-inference-framework"

	// Gateway class types
	gatewayClassIstio    = "istio"
	gatewayClassKgateway = "kgateway"

	// Test resource names
	inferenceServiceName = "pd-disaggregation-test"
	gatewayName          = "mif"

	// Kubernetes resource names
	secretNameMorehRegistry = "moreh-registry"
	secretNameECRCreds      = "moai-inference-framework-ecr-token-refresher"
	cronJobNameECRRefresher = "moai-inference-framework-ecr-token-refresher"
	jobNameECRRefresher     = "ecr-token-refresher-init-manual"

	// Helm repository
	helmRepoName = "moreh"
	helmRepoURL  = "https://moreh-dev.github.io/helm-charts"

	// Image repositories (defaults)
	imageRepoKindDefault   = "ghcr.io/llm-d/llm-d-inference-sim"
	imageTagKindDefault    = "v0.6.1"
	imageRepoDefault       = "255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm"
	imageTagDefault        = "20250915.1"
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
	tempFileMIFValues                    = "test/e2e/moai-inference-framework-values.yaml"
	tempFileHeimdallValues               = "test/e2e/heimdall-values.yaml"
	tempFileISValues                     = "test/e2e/inference-service-values.yaml"
	tempFileIstiodValues                 = "test/e2e/istiod-values.yaml"
	tempFileKgatewayValues               = "test/e2e/kgateway-values.yaml"
	tempFileInferenceServicePrefill      = "test/e2e/inference-service-prefill.yaml"
	tempFileInferenceServiceDecode      = "test/e2e/inference-service-decode.yaml"
	tempFileInferenceServiceTemplateCommon      = "test/e2e/inference-service-template-common.yaml"
	tempFileInferenceServiceTemplatePrefillMeta = "test/e2e/inference-service-template-prefill-meta.yaml"
	tempFileInferenceServiceTemplateDecodeMeta  = "test/e2e/inference-service-template-decode-meta.yaml"
	tempFileInferenceServiceTemplateDecodeProxy = "test/e2e/inference-service-template-decode-proxy.yaml"
)
