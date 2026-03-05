# moai-inference-framework

![Version: 0.0.0](https://img.shields.io/badge/Version-0.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.0](https://img.shields.io/badge/AppVersion-0.0.0-informational?style=flat-square)

Moreh Inference Framework

**Homepage:** <https://docs.moreh.io/>

## Source Code

* <https://github.com/moreh-dev/mif/tree/main/deploy/helm/moai-inference-framework>

## Requirements

> [!CAUTION]
> Prerequisite: `cert-manager` must be installed before you begin. The below dependencies will be installed automatically with this chart unless they are disabled in the `values.yaml` file.

| Repository | Name | Version |
|------------|------|---------|
| https://charts.min.io | minio | 5.4.0 |
| https://grafana.github.io/helm-charts | loki | 6.30.0 |
| https://helm.mittwald.de | replicator(kubernetes-replicator) | 2.12.2 |
| https://helm.vector.dev | vector | 0.39.0 |
| https://kedacore.github.io/charts | keda | 2.18.0 |
| https://moreh-dev.github.io/helm-charts | odin | v0.8.0-rc.1 |
| https://moreh-dev.github.io/helm-charts | odin-crd | v0.8.0-rc.1 |
| https://prometheus-community.github.io/helm-charts | prometheus-stack(kube-prometheus-stack) | 80.7.0 |
| oci://registry-1.docker.io/bitnamicharts | common | 2.31.4 |
| oci://registry.k8s.io/lws/charts | lws | 0.8.0 |
| oci://registry.k8s.io/nfd/charts | nfd(node-feature-discovery) | 0.18.3 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| commonLabels | object | `{}` | Labels applied to all resources. |
| ecrTokenRefresher.aws.accessKeyId | string | `""` | AWS_ACCESS_KEY_ID |
| ecrTokenRefresher.aws.region | string | `"ap-northeast-2"` | AWS Region. |
| ecrTokenRefresher.aws.secretAccessKey | string | `""` | AWS_SECRET_ACCESS_KEY |
| ecrTokenRefresher.enabled | bool | `true` | Enable ECR token refresher. |
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
| keda.enabled | bool | `true` | Enable kedacore/keda. Set to false if already deployed. |
| loki.backend.extraArgs[0] | string | `"-config.expand-env=true"` |  |
| loki.backend.extraEnvFrom[0].secretRef.name | string | `"loki-bucket"` |  |
| loki.backend.extraEnvFrom[1].configMapRef.name | string | `"loki-bucket"` |  |
| loki.backend.nodeSelector."node-role.kubernetes.io/control-plane" | string | `""` |  |
| loki.backend.persistence.volumeClaimsEnabled | bool | `false` |  |
| loki.backend.replicas | int | `1` |  |
| loki.enabled | bool | `true` | Enable grafana/loki. |
| loki.gateway.extraArgs[0] | string | `"-config.expand-env=true"` |  |
| loki.gateway.extraEnvFrom[0].secretRef.name | string | `"loki-bucket"` |  |
| loki.gateway.extraEnvFrom[1].configMapRef.name | string | `"loki-bucket"` |  |
| loki.gateway.nodeSelector."node-role.kubernetes.io/control-plane" | string | `""` |  |
| loki.gateway.replicas | int | `1` |  |
| loki.loki.auth_enabled | bool | `false` |  |
| loki.loki.commonConfig.replication_factor | int | `1` |  |
| loki.loki.image.tag | string | `"3.5.1"` |  |
| loki.loki.schemaConfig.configs[0].from | string | `"2024-06-24"` |  |
| loki.loki.schemaConfig.configs[0].index.period | string | `"24h"` |  |
| loki.loki.schemaConfig.configs[0].index.prefix | string | `"loki_index_"` |  |
| loki.loki.schemaConfig.configs[0].object_store | string | `"s3"` |  |
| loki.loki.schemaConfig.configs[0].schema | string | `"v13"` |  |
| loki.loki.schemaConfig.configs[0].store | string | `"tsdb"` |  |
| loki.loki.storage.bucketNames.admin | string | `"loki"` |  |
| loki.loki.storage.bucketNames.chunks | string | `"loki"` |  |
| loki.loki.storage.bucketNames.ruler | string | `"loki"` |  |
| loki.loki.storage.s3.accessKeyId | string | `"${AWS_ACCESS_KEY_ID}"` |  |
| loki.loki.storage.s3.endpoint | string | `"http://${BUCKET_HOST}:${BUCKET_PORT}"` |  |
| loki.loki.storage.s3.region | string | `"${BUCKET_REGION}"` |  |
| loki.loki.storage.s3.s3ForcePathStyle | bool | `true` |  |
| loki.loki.storage.s3.secretAccessKey | string | `"${AWS_SECRET_ACCESS_KEY}"` |  |
| loki.loki.storage_config.tsdb_shipper.active_index_directory | string | `"/var/loki/tsdb-index"` |  |
| loki.loki.storage_config.tsdb_shipper.cache_location | string | `"/var/loki/tsdb-cache"` |  |
| loki.loki.storage_config.tsdb_shipper.cache_ttl | string | `"168h"` |  |
| loki.loki.structuredConfig.compactor.delete_request_store | string | `"s3"` |  |
| loki.loki.structuredConfig.compactor.retention_enabled | bool | `true` |  |
| loki.loki.structuredConfig.limits_config.ingestion_burst_size_mb | int | `60` |  |
| loki.loki.structuredConfig.limits_config.ingestion_rate_mb | int | `30` |  |
| loki.loki.structuredConfig.limits_config.max_entries_limit_per_query | int | `50000` |  |
| loki.loki.structuredConfig.limits_config.max_query_series | int | `10000` |  |
| loki.loki.structuredConfig.limits_config.per_stream_rate_limit | string | `"30MB"` |  |
| loki.loki.structuredConfig.limits_config.per_stream_rate_limit_burst | string | `"60MB"` |  |
| loki.loki.structuredConfig.limits_config.retention_period | string | `"2160h"` |  |
| loki.loki.structuredConfig.limits_config.split_queries_by_interval | string | `"24h"` |  |
| loki.read.extraArgs[0] | string | `"-config.expand-env=true"` |  |
| loki.read.extraEnvFrom[0].secretRef.name | string | `"loki-bucket"` |  |
| loki.read.extraEnvFrom[1].configMapRef.name | string | `"loki-bucket"` |  |
| loki.read.nodeSelector."node-role.kubernetes.io/control-plane" | string | `""` |  |
| loki.read.replicas | int | `1` |  |
| loki.write.extraArgs[0] | string | `"-config.expand-env=true"` |  |
| loki.write.extraEnvFrom[0].secretRef.name | string | `"loki-bucket"` |  |
| loki.write.extraEnvFrom[1].configMapRef.name | string | `"loki-bucket"` |  |
| loki.write.nodeSelector."node-role.kubernetes.io/control-plane" | string | `""` |  |
| loki.write.persistence.volumeClaimsEnabled | bool | `false` |  |
| loki.write.replicas | int | `1` |  |
| lokiBucket.accessKey | string | `""` | MinIO access key for Loki storage. Defaults to minio.rootUser. |
| lokiBucket.host | string | `""` | MinIO service host for Loki storage. Defaults to <release>-minio. Use the FQDN (e.g. minio.minio.svc.cluster.local) for cross-namespace access. |
| lokiBucket.secretKey | string | `""` | MinIO secret key for Loki storage. Defaults to minio.rootPassword. |
| lws.enabled | bool | `true` | Enable kubernetes-sigs/lws. Set to false if already deployed. |
| minio.buckets[0].name | string | `"loki"` |  |
| minio.enabled | bool | `true` | Enable minio/minio as the S3-compatible object storage backend for Loki. Set to false if MinIO is already deployed; in that case, configure loki storage to point to the existing MinIO service. |
| minio.mode | string | `"standalone"` |  |
| minio.persistence.enabled | bool | `false` |  |
| minio.resources.requests.memory | string | `"2Gi"` |  |
| minio.rootPassword | string | `"minio123!"` | MinIO root password. Override with a strong password in production. |
| minio.rootUser | string | `"minio"` | MinIO root user. |
| nameOverride | string | `""` | Chart name override. |
| namespaceOverride | string | `""` | Namespace override. |
| nfd.enabled | bool | `true` | Enable kubernetes-sigs/node-feature-discovery. Set to false if already deployed. |
| nfd.worker.tolerations | list | `[{"effect":"NoSchedule","key":"amd.com/gpu","operator":"Exists"},{"effect":"NoSchedule","key":"nvidia.com/gpu","operator":"Exists"},{"effect":"NoExecute","key":"amd-dcm","operator":"Equal","value":"up"},{"effect":"NoSchedule","key":"amd-gpu-unhealthy","operator":"Exists"}]` | NFD Worker Tolerations to allow NFD workers to deploy to GPU nodes |
| odin-crd.enabled | bool | `true` | Enable moreh/odin CRD. Set to false if already deployed. |
| odin.enabled | bool | `true` | Enable moreh/odin. Set to false if already deployed. |
| odin.image.pullSecrets[0].name | string | `"moreh-registry"` |  |
| prometheus-stack.alertmanager.enabled | bool | `false` |  |
| prometheus-stack.coreDns.enabled | bool | `false` |  |
| prometheus-stack.defaultRules.create | bool | `false` |  |
| prometheus-stack.enabled | bool | `true` | Enable prometheus-community/kube-prometheus-stack. Set to false if already deployed. |
| prometheus-stack.grafana.enabled | bool | `true` |  |
| prometheus-stack.grafana.sidecar.dashboards.enabled | bool | `true` |  |
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
| vector.customConfig.api.address | string | `"0.0.0.0:8686"` |  |
| vector.customConfig.api.enabled | bool | `true` |  |
| vector.customConfig.data_dir | string | `"/vector-data"` |  |
| vector.customConfig.sinks.loki.encoding.codec | string | `"json"` |  |
| vector.customConfig.sinks.loki.endpoint | string | `"http://{{ .Release.Name }}-loki-gateway"` |  |
| vector.customConfig.sinks.loki.inputs[0] | string | `"mif_log_transform"` |  |
| vector.customConfig.sinks.loki.labels.app | string | `"{{`{{ app }}`}}"` |  |
| vector.customConfig.sinks.loki.labels.inference_service | string | `"{{`{{ inference_service }}`}}"` |  |
| vector.customConfig.sinks.loki.labels.level | string | `"{{`{{ level }}`}}"` |  |
| vector.customConfig.sinks.loki.labels.namespace | string | `"{{`{{ namespace }}`}}"` |  |
| vector.customConfig.sinks.loki.labels.node_name | string | `"{{`{{ node_name }}`}}"` |  |
| vector.customConfig.sinks.loki.labels.pool_name | string | `"{{`{{ pool_name }}`}}"` |  |
| vector.customConfig.sinks.loki.labels.role | string | `"{{`{{ role }}`}}"` |  |
| vector.customConfig.sinks.loki.type | string | `"loki"` |  |
| vector.customConfig.sources.mif_logs.extra_label_selector | string | `"mif.moreh.io/log.collect=true"` |  |
| vector.customConfig.sources.mif_logs.type | string | `"kubernetes_logs"` |  |
| vector.customConfig.transforms.mif_log_transform.inputs[0] | string | `"mif_logs"` |  |
| vector.customConfig.transforms.mif_log_transform.source | string | `".namespace          = .kubernetes.pod_namespace\n.node_name          = \"$VECTOR_SELF_NODE_NAME\"\n.app                = get(.kubernetes.pod_labels, [\"app.kubernetes.io/name\"])      ?? \"\"\n.inference_service  = get(.kubernetes.pod_labels, [\"app.kubernetes.io/instance\"])  ?? \"\"\n.pool_name          = get(.kubernetes.pod_labels, [\"mif.moreh.io/pool\"])           ?? \"\"\n.role               = get(.kubernetes.pod_labels, [\"mif.moreh.io/role\"])           ?? \"\"\n\nlog_format = get(.kubernetes.pod_labels, [\"mif.moreh.io/log.format\"]) ?? \"\"\n\nif log_format == \"json\" {\n  structured, err = parse_json(.message)\n  if err == null {\n    . = merge!(., structured)\n    msg, err = get(., [\"msg\"])\n    if err == null {\n      .message = msg\n      del(.msg)\n    }\n    time, err = get(., [\"time\"])\n    if err == null {\n      .timestamp = time\n      del(.time)\n    }\n  }\n}\n\ndel(.file)\ndel(.source_type)\ndel(.stream)\ndel(.kubernetes)\n"` |  |
| vector.customConfig.transforms.mif_log_transform.type | string | `"remap"` |  |
| vector.enabled | bool | `true` | Enable vector/vector as a DaemonSet log collector. |
| vector.role | string | `"Agent"` |  |
| vector.tolerations[0].effect | string | `"NoExecute"` |  |
| vector.tolerations[0].key | string | `"node.kubernetes.io/unschedulable"` |  |
| vector.tolerations[0].operator | string | `"Exists"` |  |
| vector.tolerations[0].tolerationSeconds | int | `5` |  |
| vector.tolerations[1].effect | string | `"NoSchedule"` |  |
| vector.tolerations[1].key | string | `"node-role.kubernetes.io/compute"` |  |
| vector.tolerations[1].operator | string | `"Equal"` |  |
| vector.tolerations[1].value | string | `"true"` |  |
| vector.tolerations[2].effect | string | `"NoSchedule"` |  |
| vector.tolerations[2].key | string | `"amd.com/gpu"` |  |
| vector.tolerations[2].operator | string | `"Exists"` |  |
| vector.updateStrategy.rollingUpdate.maxUnavailable | int | `10` |  |
| vector.updateStrategy.type | string | `"RollingUpdate"` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
