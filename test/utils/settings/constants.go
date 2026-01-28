//go:build e2e
// +build e2e

package settings

import "time"

// Kind Cluster
const (
	KindClusterName = "mif-e2e"
)

// Cert Manager
const (
	CertManagerHelmRepoURL = "oci://quay.io/jetstack/charts/cert-manager"
	CertManagerVersion     = "v1.18.4"
	CertManagerNamespace   = "cert-manager"
)

// Gateway API
const (
	GatewayAPIYAML                   = "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml"
	GatewayAPIInferenceExtensionYAML = "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml"
)

// Istio
const (
	IstioHelmRepoURL     = "https://istio-release.storage.googleapis.com/charts"
	IstioVersion         = "1.28.1"
	IstioNamespace       = "istio-system"
	IstiodValuesFile     = "test/utils/config/istiod-values.yaml"
	IstioGatewayTemplate = "test/utils/config/gateway-istio.yaml.tmpl"
)

// Kgateway
const (
	KgatewayCrdsHelmRepoURL = "oci://cr.kgateway.dev/kgateway-dev/charts/kgateway-crds"
	KgatewayHelmRepoURL     = "oci://cr.kgateway.dev/kgateway-dev/charts/kgateway"
	KgatewayCrdsVersion     = "v2.1.1"
	KgatewayVersion         = "v2.1.1"
	KgatewayNamespace       = "kgateway-system"
	KgatewayValuesFile      = "test/utils/config/kgateway-values.yaml"
	KgatewayGatewayTemplate = "test/utils/config/gateway-kgateway.yaml.tmpl"
)

// MIF
const (
	MorehHelmRepoURL  = "https://moreh-dev.github.io/helm-charts"
	MIFValuesTemplate = "test/utils/config/mif-values.yaml.tmpl"
)

// Heimdall
const (
	HeimdallVersion = "v0.6.1"
)

// Inference service
const (
	InferenceServiceName                = "pd"
	InferenceServicePrefill             = "test/config/base/resources/inference-service-prefill.yaml.tmpl"
	InferenceServiceDecode              = "test/config/base/resources/inference-service-decode.yaml.tmpl"
	InferenceServiceTemplateCommon      = "test/config/base/resources/inference-service-template-common.yaml.tmpl"
	InferenceServiceTemplatePrefillMeta = "test/config/base/resources/inference-service-template-prefill-meta.yaml.tmpl"
	InferenceServiceTemplateDecodeMeta  = "test/config/base/resources/inference-service-template-decode-meta.yaml.tmpl"
	InferenceServiceTemplateDecodeProxy = "test/config/base/resources/inference-service-template-decode-proxy.yaml.tmpl"
	GatewayName                         = "mif"
	MorehRegistrySecretName             = "moreh-registry"
	ImageRepoKindDefault                = "ghcr.io/llm-d/llm-d-inference-sim"
	ImageTagKindDefault                 = "v0.6.1"
	ImageRepoDefault                    = "255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm"
	ImageTagDefault                     = "20250915.1"
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
