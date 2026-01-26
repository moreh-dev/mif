//go:build e2e
// +build e2e

package utils

import "time"

// Resource names used in E2E tests
const (
	// Helm release names
	helmReleaseMIF = "mif"

	// Gateway class types
	gatewayClassIstio    = "istio"
	gatewayClassKgateway = "kgateway"

	// Test resource names
	inferenceServiceName = "pd-disaggregation-test"
	gatewayName          = "mif"

	// Kubernetes resource names
	secretNameMorehRegistry = "moreh-registry"

	// Helm repository
	helmRepoName = "moreh"
	helmRepoURL  = "https://moreh-dev.github.io/helm-charts"

	// Image repositories (defaults)
	imageRepoKindDefault = "ghcr.io/llm-d/llm-d-inference-sim"
	imageTagKindDefault  = "v0.6.1"
	imageRepoDefault     = "255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm"
	imageTagDefault      = "20250915.1"
)

// Inference perf experiment defaults
const (
	inferencePerfS3PrefixBase = "vllm"
	inferencePerfPreset       = "workertemplate-vllm-common"
	inferencePerfExpType      = "performance"
	inferencePerfExpName      = "synthetic_random_i1024_o1024_c64"
)

// Timeout and interval constants for E2E tests
const (
	TimeoutShort    = 2 * time.Minute
	TimeoutMedium   = 5 * time.Minute
	TimeoutLong     = 10 * time.Minute
	TimeoutVeryLong = 15 * time.Minute

	IntervalShort  = 2 * time.Second
	IntervalMedium = 5 * time.Second
	IntervalLong   = 10 * time.Second
)

// File paths for temporary test files
const (
	tempFileMIFValues                           = "test/tmp/moai-inference-framework-values.yaml"
	tempFileHeimdallValues                      = "test/tmp/heimdall-values.yaml"
	tempFileISValues                            = "test/tmp/inference-service-values.yaml"
	tempFileIstiodValues                        = "test/tmp/istiod-values.yaml"
	tempFileKgatewayValues                      = "test/tmp/kgateway-values.yaml"
	tempFileInferenceServicePrefill             = "test/tmp/inference-service-prefill.yaml"
	tempFileInferenceServiceDecode              = "test/tmp/inference-service-decode.yaml"
	tempFileInferenceServiceTemplateCommon      = "test/tmp/inference-service-template-common.yaml"
	tempFileInferenceServiceTemplatePrefillMeta = "test/tmp/inference-service-template-prefill-meta.yaml"
	tempFileInferenceServiceTemplateDecodeMeta  = "test/tmp/inference-service-template-decode-meta.yaml"
	tempFileInferenceServiceTemplateDecodeProxy = "test/tmp/inference-service-template-decode-proxy.yaml"
)
