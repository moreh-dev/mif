//go:build e2e
// +build e2e

package settings

import "time"

// Cert Manager
const (
	CertManagerVersion   = "v1.18.4"
	CertManagerNamespace = "cert-manager"
)

// MIF
const (
	MIFValuesTemplate = "test/utils/config/mif-values.yaml.tmpl"
)

// Gateway API
const (
	GatewayAPIYAML                   = "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml"
	GatewayAPIInferenceExtensionYAML = "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml"
)

// Istio
const (
	IstioVersion         = "1.28.5"
	IstioNamespace       = "istio-system"
	IstiodValuesFile     = "test/utils/config/istiod-values.yaml"
	IstioGatewayTemplate = "test/utils/config/gateway-istio.yaml.tmpl"
)

// Kgateway
const (
	KgatewayCrdsVersion     = "v2.1.1"
	KgatewayVersion         = "v2.1.1"
	KgatewayNamespace       = "kgateway-system"
	KgatewayValuesFile      = "test/utils/config/kgateway-values.yaml"
	KgatewayGatewayTemplate = "test/utils/config/gateway-kgateway.yaml.tmpl"
)

// Models
const (
	ModelPV  = "test/utils/config/model-pv.yaml.tmpl"
	ModelPVC = "test/utils/config/model-pvc.yaml.tmpl"
)

// Timeout and interval constants for E2E tests
const (
	TimeoutShort    = 2 * time.Minute
	TimeoutMedium   = 5 * time.Minute
	TimeoutLong     = 10 * time.Minute
	TimeoutVeryLong = 15 * time.Minute
	Timeout30Min    = 30 * time.Minute

	IntervalShort  = 2 * time.Second
	IntervalMedium = 5 * time.Second
	IntervalLong   = 10 * time.Second
)
