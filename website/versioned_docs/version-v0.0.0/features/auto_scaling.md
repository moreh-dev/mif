---
sidebar_position: 6

title: 'Auto-scaling'
---

# Auto-scaling

The model inference endpoints provided by the MoAI Inference Fraemwork are often just one of many functions running on the overall AI compute infrastructure. Therefore, it is essential to allocate the appropriate amount of GPU resources (to run the appropriate number of Pods) so that GPUs are not under-utilized while still handling all incoming traffic and meeting the defined service level objectives (SLOs).

This where auto-scaling comes into play. Instead of allocating all GPU resources from the start, the system begins with a small number of Pods, and adds more only when traffic increases or SLOs are at risk. Additionally, if traffic decreases, the number of Pods is reduced accordingly. It is also necessary to adjust not only the total number of Pods but also the number of Pods assigned to each disaggregated parts (prefill, decode, a set of experts, etc.).

## Key features

- The framework can dynamically adjust the number of GPU resources (the number of Pods) according to the given SLOs and the current amount of traffic.
- Users can manually configure auto-scaling rules using [KEDA](https://keda.sh/).

---

## Manual configuration of auto-scaling rules

### Installing KEDA

You can install KEDA as follows. See [KEDA / Deploying KEDA](https://keda.sh/docs/deploy/) for more details.

```shell
helm repo add kedacore https://kedacore.github.io/charts
helm repo update kedacore
helm upgrade -i keda kedacore/keda \
    --version 2.18.0 \
    -n keda \
    --create-namespace
```

### Enabling auto-scaling in the Odin inference service

To enable auto-scaling, set the `autoscaling.enabled` to `true` under each profile (`decode` and `prefill`) of the **Odin** inference service. If this is set to `false`, the number of replicas remains fixed as specified in the `replicas` field. However, if it is set to `true`, the number of replicas dynamically changes between `minReplicaCount` and `maxReplicaCount`. The following is an example of configuring auto-scaling for the prefill phase (and not for the decode phase).

```yaml inference-service-values.yaml
...
decode:
  replicas: 4
  autoscaling:
    enabled: false
  ...

prefill:
  autoscaling:
    enabled: true
    minReplicaCount: 1
    maxReplicaCount: 6
    behavior:
      scaleUp:
        stabilizationWindowSeconds: 100
        policies:
          - type: Pods
            value: 1
            periodSeconds: 200
      scaleDown:
        stabilizationWindowSeconds: 200
        policies:
          - type: Pods
            value: 1
            periodSeconds: 60
    triggers:
      - type: prometheus
        metricType: Value
        metadata:
          serverAddress: http://prometheus-operated.prometheus-stack:9090
          query: histogram_quantile(0.9, sum by(le) (rate(llm_time_to_first_token_seconds_bucket{job="{{ include "common.names.namespace" . }}/{{ include "inferenceService.prefill.fullname" . }}"}[2m])))
          threshold: "1"  # Scale up if the P90 TTFT exceeds 1 second
  ...
```

### Configuration parameters

#### Scaling behavior

The `behavior.scaleUp` and `behavior.scaleDown` section control how fast the system scales up and down, respectively. For more details, see the [Kubernetes HPA Scale Velocity KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/853-configurable-hpa-scale-velocity/README.md).

- `stabilizationWindowSeconds`: specifies the duration of the time window the autoscaler considers when determining the target replica count.
  - When scaling up, the system picks the **smallest** replica count recommended during the window.
    - Set to 0 to respond immediately to load increases.
    - Use a non-zero value (e.g., 300 seconds) to prevent rapid scaling due to temporary traffic spikes.
  - When scaling down, the system picks the **largest** replica count recommended during the window.
    - Set to 0 for immediate scale-down (not recommended for production).
    - Use a non-zero value (e.g., 300 seconds) to prevent rapid scaling due to temporary traffic drops.
- `policies`: define the maximum rate at which replicas can be added or removed. Each item includes the following fields.
  - `type`: either `Pods` (absolute replica count) or `Percent` (percentage relative to the current replica count).
  - `value`: maximum number of pods or percentage that can be added or removed.
  - `periodSeconds`: the time period over which the policy applies.
- `selectPolicy`: defines how the system determines which policy to apply when multiple policies are specified.
  - Set to `Max` (default) to select the policy that allows the maximum change.
  - Set to `Min` to select the policy that allows the minimum change.
  - Set to `Disabled` to selectively disable scaling up or scaling down.

#### Prometheus metric triggers

The `triggers` section defines the metrics that will be monitored to determine when to scale. Specifically, the MoAI Inference Framework uses KEDA's Prometheus scaler. For more details, see [KEDA / Prometheus Scaler](https://keda.sh/docs/scalers/prometheus/). Each trigger has the following fields.

- `type`: `prometheus`
- `metricType`: specifies how the metric value is interpreted. It can be either `Value` or `AverageValue`.
  - Set to `Value` to adjust the replica count so that `currentMetricValue` equals `threshold`.
    - `desiredReplicaCount = currentReplicaCount * (currentMetricValue / threshold)`
  - Set to `AverageValue` to set the number of replicas to `currentMetricValue / threshold`.
    - `desiredReplicaCount = currentMetricValue / threshold`
- `metadata`: defines the metric value and the threshold.
  - `serverAddress`: Prometheus server endpoint.
  - `query`: a PromQL query to calculate the metric value.
  - `threshold`: the target value that triggers scaling.

### Available metrics for auto-scaling

The following metrics can be used to configure auto-scaling triggers. These metrics are originally exposed by vLLM with the `vllm:` prefix, but are relabeled with the `llm_` prefix in Prometheus through the PodMonitor configuration (for example, `vllm:time_to_first_token_seconds` becomes `llm_time_to_first_token_seconds`) to maintain compatibility with other inference engines such as SGLang.

#### Latency metrics

- `llm_time_to_first_token_seconds_bucket`: a histogram of time to first token (TTFT) in seconds, used to scale based on how quickly users receive the first token.
  - Example: scale up when the P90 TTFT exceeds 1 second.

```yaml
metricType: Value
metadata:
  query: histogram_quantile(0.9, sum by(le) (rate(llm_time_to_first_token_seconds_bucket{job="{{ include "common.names.namespace" . }}/{{ include "inferenceService.prefill.fullname" . }}"}[2m])))
  threshold: '1'
```

- `llm_inter_token_latency_seconds_bucket`: a histogram of inter-token latency (ITL) in seconds, used to scale based on token generation speed.
  - Example: scale up when the P90 ITL exceeds 200 ms.

```yaml
metricType: Value
metadata:
  query: histogram_quantile(0.9, sum by(le) (rate(llm_inter_token_latency_seconds_bucket{job="{{ include "common.names.namespace" . }}/{{ include "inferenceService.prefill.fullname" . }}"}[2m])))
  threshold: '0.2'
```

- `llm_e2e_request_latency_seconds_bucket`: a histogram of end-to-end request latency (E2EL) in seconds, used to scale based on total request processing time.
  - Example: scale up when the P95 E2EL exceeds 5 seconds.

```yaml
metricType: Value
metadata:
  query: histogram_quantile(0.95, sum by(le) (rate(llm_e2e_request_latency_seconds_bucket{job="{{ include "common.names.namespace" . }}/{{ include "inferenceService.prefill.fullname" . }}"}[2m])))
  threshold: '5'
```

#### Queue and load-related metrics

- `llm_num_requests_waiting`: the number of requests waiting to be processed, used to scale up when too many requests are queued.

#### Resource-related metrics

- `llm_kv_cache_usage_perc`: the KV cache usage (1 = 100% utilization), used to scale up when the cache is nearly full.
