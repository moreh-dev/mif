# moai-inference-framework

![Version: 0.0.0](https://img.shields.io/badge/Version-0.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.0](https://img.shields.io/badge/AppVersion-0.0.0-informational?style=flat-square)

Moreh Inference Framework

**Homepage:** <https://github.com/moreh-dev/mif>

## Source Code

* <https://github.com/moreh-dev/mif/tree/main/deploy/helm/moai-inference-framework>

## Requirements

> [!CAUTION]
> Prerequisite: `cert-manager` must be installed before you begin. The below dependencies will be installed automatically with this chart unless they are disabled in the `values.yaml` file.

| Repository | Name | Version |
|------------|------|---------|
| https://helm.mittwald.de | replicator(kubernetes-replicator) | 2.12.2 |
| https://moreh-dev.github.io/helm-charts | odin | v0.3.0 |
| https://moreh-dev.github.io/helm-charts | odin-crd | v0.3.0 |
| https://prometheus-community.github.io/helm-charts | prometheus-stack(kube-prometheus-stack) | 80.7.0 |
| oci://registry-1.docker.io/bitnamicharts | common | 2.31.4 |
| oci://registry.k8s.io/lws/charts | lws | 0.7.0 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| commonLabels | object | `{}` | Labels applied to all resources. |
| ecrTokenRefresher.aws.accessKeyId | string | `""` | AWS_ACCESS_KEY_ID |
| ecrTokenRefresher.aws.region | string | `"ap-northeast-2"` | AWS Region. |
| ecrTokenRefresher.aws.secretAccessKey | string | `""` | AWS_SECRET_ACCESS_KEY |
| ecrTokenRefresher.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy. |
| ecrTokenRefresher.image.pullSecrets | list | `[]` | Image pull secrets. |
| ecrTokenRefresher.image.repository | string | `"heyvaldemar/aws-kubectl"` | Image repository. |
| ecrTokenRefresher.image.tag | string | `"58dad7caa5986ceacd1bc818010a5e132d80452b"` | Image tag (defaults to chart appVersion if not set). |
| ecrTokenRefresher.pullSecret.annotations | object | `{"replicator.v1.mittwald.de/replicate-to-matching":"mif=enabled"}` | Annotations to add to the generated secret. |
| ecrTokenRefresher.pullSecret.name | string | `"moreh-registry"` | Name of the container registry secret to create or update. |
| ecrTokenRefresher.pullSecret.server | string | `"255250787067.dkr.ecr.ap-northeast-2.amazonaws.com"` | URL of the container registry. |
| ecrTokenRefresher.pullSecret.username | string | `"AWS"` | Username for the container registry access. |
| ecrTokenRefresher.schedule | string | `"0 */6 * * *"` | Cron schedule in standard cron format. |
| ecrTokenRefresher.serviceAccount.annotations | object | `{}` | Annotations added to the ServiceAccount. |
| ecrTokenRefresher.serviceAccount.automount | bool | `true` | Whether to automatically mount API credentials. |
| ecrTokenRefresher.serviceAccount.create | bool | `true` | Whether to create a ServiceAccount. |
| ecrTokenRefresher.serviceAccount.name | string | `""` | Name of the ServiceAccount (auto-generated if empty and create is true). |
| fullnameOverride | string | `""` | Full name override. |
| global | object | `{"imagePullSecrets":[]}` | global values are shared across all sub-charts if the value's key matches. |
| global.imagePullSecrets | list | `[]` | Image pull secrets. |
| lws.enabled | bool | `true` | Enable kubernetes-sigs/lws. Set to false if already deployed. |
| nameOverride | string | `""` | Chart name override. |
| namespaceOverride | string | `""` | Namespace override. |
| odin-crd.enabled | bool | `true` | Enable moreh/odin CRD. Set to false if already deployed. |
| odin.enabled | bool | `true` | Enable moreh/odin. Set to false if already deployed. |
| odin.image.pullSecrets[0].name | string | `"moreh-registry"` |  |
| prometheus-stack.alertmanager.enabled | bool | `false` |  |
| prometheus-stack.coreDns.enabled | bool | `false` |  |
| prometheus-stack.defaultRules.create | bool | `false` |  |
| prometheus-stack.enabled | bool | `true` | Enable prometheus-community/kube-prometheus-stack. Set to false if already deployed. |
| prometheus-stack.grafana.enabled | bool | `true` |  |
| prometheus-stack.kubeApiServer.enabled | bool | `false` |  |
| prometheus-stack.kubeControllerManager.enabled | bool | `false` |  |
| prometheus-stack.kubeDns.enabled | bool | `false` |  |
| prometheus-stack.kubeEtcd.enabled | bool | `false` |  |
| prometheus-stack.kubeProxy.enabled | bool | `false` |  |
| prometheus-stack.kubeScheduler.enabled | bool | `false` |  |
| prometheus-stack.kubeStateMetrics.enabled | bool | `true` |  |
| prometheus-stack.kubelet.enabled | bool | `true` |  |
| prometheus-stack.kubernetesServiceMonitors.enabled | bool | `true` |  |
| prometheus-stack.nodeExporter.enabled | bool | `true` |  |
| prometheus-stack.prometheus.enabled | bool | `true` |  |
| prometheus-stack.prometheusOperator.enabled | bool | `true` |  |
| prometheus-stack.thanosRuler.enabled | bool | `false` |  |
| prometheus-stack.windowsMonitoring.enabled | bool | `false` |  |
| replicator.enabled | bool | `true` | Enable mittwald/kubernetes-replicator. Set to false if already deployed. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
