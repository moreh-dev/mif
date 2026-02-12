---
sidebar_position: 1

title: 'DeepSeek R1 671B on AMD MI300X GPUs: Maximum throughput'
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# DeepSeek R1 671B on AMD MI300X GPUs: Maximum throughput

This article presents the performance evaluation method and results of **DeepSeek R1 671B** inference on 5x AMD MI300X servers (40 GPUs in total).

## Overview

The purpose of this benchmarking is to measure the maximum throughput (output tokens/sec) achievable when running distributed inference of the DeepSeek R1 671B model on a 5-node AMD MI300X GPU cluster. This metric directly determines the cost efficiency of inference service (tokens/$). This benchmarking demonstrates three key points:

- We built a distributed inference system operating at the AMD GPU cluster level **in real deployments**, which efficiently handles high-concurrency requests via prefill-decode disaggregation and expert parallelism.
- MoAI Inference Framework delivers industry-leading throughput on AMD MI300X GPU clusters, which enables lower cost-per-token ($/token) configurations.
- MoAI Inference Framework achieves throughput on AMD MI300X GPU clusters that is on par with what is attainable on NVIDIA H100 GPU clusters.

The experimental methodology was largely designed by referring to the following report from the SGLang team, which measures the performance of PD disaggregation and expert parallelism on an NVIDIA H100 GPU cluster. The key difference is that, while the SGLang team measures prefill-only and decode-only performance separately, our benchmarking integrates prefill and decode instances and measures performance in an end-to-end inference environment, which more accurately reflects real-world achievable performance.

- Reference: [Deploying DeepSeek with PD Disaggregation and Large-Scale Expert Parallelism on 96 H100 GPUs](https://lmsys.org/blog/2025-05-05-large-scale-ep/)

---

## Target environment and configuration

| Item              | Description                                       |
| ----------------- | ------------------------------------------------- |
| GPU servers       | 5x servers, each equipped with 8x AMD MI300X GPUs |
| Networking        | InfiniBand HDR                                    |
| Inference engine  | Moreh vLLM (0.11.0rc2.moreh20251212)              |
| Model             | `deepseek-ai/DeepSeek-R1`                         |
| PD disaggregation | 2x prefill, 3x decode instances                   |
| Parallelization   | EP=8 + DP=8                                       |

The specifications of each GPU server are as follows:

- CPU: 2x AMD EPYC 9474F 48-core 3.6 GHz
- Main memory: 2,304 GB
- GPU: 8x AMD Instinct MI300X OAM GPU 192 GB
- Server: Gigabyte G593-ZX1-AAX1
- Operating system: Ubuntu 22.04.4 LTS
- ROCm version: 6.4.1

---

## Deployment

Please make sure to install all [prerequisites](/getting_started/prerequisites) before starting this benchmarking. Also, please refer to the [quickstart](/getting_started/quickstart) to understand how to run the MoAI Inference Framework.

In this benchmarking, you need to deploy the **Istio** gateway, the **Heimdall** scheduler configured to specify the basic routing strategy for PD disaggregation, and the **Odin** inference service configured to run two prefill instances and three decode instances across five GPU servers using optimized settings.

First, you need to have a namespace for deploying and running the components of the MoAI Inference Framework. In this guide, we assume the namespace is named `mif`.

```shell
kubectl create namespace mif
```

**AWS credentials must be configured in this namespace to allow the container images of the MoAI Inference Framework to be downloaded**. For details, refer to the "Amazon ECR token for Moreh's container image repository" section in the [prerequisites](/getting_started/prerequisites).

Then, you can use the following configuration files for the components. Click to view their contents. **You must store the DeepSeek-R1 model checkpoint on the host of every worker node and specify its path on line 19 of the `inference-service-values.yaml` file**. This path will be mounted to `/app/model/DeepSeek-R1` inside the pod and used to run the Moreh vLLM server.

<Tabs>
<TabItem value="istio" label="Istio gateway configuration (gateway.yaml)" default>

```yaml gateway.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mif-gateway-infrastructure
  namespace: mif
data:
  service: |
    spec:
      type: ClusterIP
  deployment: |
    spec:
      template:
        metadata:
          annotations:
            proxy.istio.io/config: |
              accessLogFile: /dev/stdout
              accessLogEncoding: JSON
        spec:
          containers:
            - name: istio-proxy
              resources:
                limits: null

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
  namespace: mif
spec:
  gatewayClassName: istio
  infrastructure:
    parametersRef:
      group: ''
      kind: ConfigMap
      name: mif-gateway-infrastructure
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
```

</TabItem>
<TabItem value="heimdall" label="Heimdall scheduler configuration (heimdall-values.yaml)">

```yaml heimdall-values.yaml
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: pd-profile-handler
    - type: prefill-filter
    - type: decode-filter
    - type: active-request-scorer
      parameters:
        requestTimeout: '20m'
    - type: max-score-picker
    - type: random-picker
  schedulingProfiles:
    - name: prefill
      plugins:
        - pluginRef: prefill-filter
        - pluginRef: active-request-scorer
          weight: 1
        - pluginRef: max-score-picker
    - name: decode
      plugins:
        - pluginRef: decode-filter
        - pluginRef: active-request-scorer
          weight: 1
        - pluginRef: max-score-picker

tolerations:
  - key: amd.com/gpu
    operator: Exists
    effect: NoSchedule

gateway:
  name: mif
  gatewayClassName: istio
```

</TabItem>
<TabItem value="odin" label="Odin inference service configuration (inference-service-values.yaml)">

```yaml inference-service-values.yaml {19}
global:
  imagePullSecrets:
    - name: moreh-registry

extraVolumeMounts:
  - name: shm
    mountPath: /dev/shm
  - name: dsr1
    mountPath: /app/model/DeepSeek-R1
    readOnly: false

extraVolumes:
  - name: shm
    emptyDir:
      medium: Memory
      sizeLimit: 16Gi
  - name: dsr1
    hostPath:
      path: /path/to/deepseek-r1

_common: &common
  image:
    repository: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm
    tag: vllm_251212

  updateStrategy:
    type: Recreate

  resources:
    requests: &resources
      amd.com/gpu: '8'
      mellanox/hca: '1'
    limits: *resources

  tolerations:
    - key: amd.com/gpu
      operator: Exists
      effect: NoSchedule

  podMonitor:
    labels:
      prometheus-stack/prometheus: enabled

extraEnvVars:
  - name: UCX_IB_PCI_RELAXED_ORDERING
    value: 'on'
  - name: UCX_TLS
    value: rocm_copy,rocm_ipc,self,sm,rc_x
  - name: NCCL_IB_PCI_RELAXED_ORDERING
    value: '1'
  - name: NCCL_NET_GDR_LEVEL
    value: '3'
  - name: NCCL_MIN_NCHANNELS
    value: '112'
  - name: VLLM_RANDOMIZE_DP_DUMMY_INPUTS
    value: '1'
  - name: VLLM_ROCM_USE_AITER
    value: '1'
  - name: VLLM_ROCM_USE_AITER_FP8BMM
    value: '0'
  - name: VLLM_ALL2ALL_BACKEND
    value: 'mori'
  - name: VLLM_HTTP_TIMEOUT_KEEP_ALIVE
    value: '1000000000'
  - name: VLLM_NIXL_ABORT_REQUEST_TIMEOUT
    value: '1000000000'
  - name: VLLM_NIXL_SIDE_CHANNEL_HOST
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
  - name: VLLM_LOG_STATS_INTERVAL
    value: '10'
  - name: VLLM_SERVERSIDE_LOGGING
    value: '1'
  - name: VLLM_SERVERSIDE_LOG_INTERVAL
    value: '10'
  - name: GLOO_SOCKET_IFNAME
    value: ''
  - name: NCCL_SOCKET_IFNAME
    value: ''
  - name: TP_SOCKET_IFNAME
    value: ''

proxy:
  image:
    tag: c8abd08

decode:
  replicas: 3

  <<: *common

  parallelism:
    data: 8

  extraArgs:
    - /app/model/DeepSeek-R1
    - --served-model-name
    - deepseek-ai/DeepSeek-R1
    - --trust-remote-code
    - --no-enable-prefix-caching
    - --no-enable-chunked-prefill
    - --enforce-eager
    - --tensor-parallel-size
    - '1'
    - --enable-expert-parallel
    - --max-model-len
    - '8192'
    - --max-num-seqs
    - '2048'
    - --kv-cache-dtype
    - fp8_e4m3
    - --quantization
    - ds_fp8_per_token
    - --block-size
    - '16'
    - --kv-transfer-config
    - '{"kv_connector":"NixlConnector","kv_role":"kv_both"}'
    - --disable-uvicorn-access-log
    - --no-enable-log-requests
    - --disable-log-stats
    - --max-num-batched-token
    - '16384'
    - --gpu-memory-utilization
    - '0.92'

  extraEnvVars:
    - name: VLLM_MOE_DP_CHUNK_SIZE
      value: '512'
    - name: VLLM_V1_OUTPUT_PROC_CHUNK_SIZE
      value: '512'
    - name: VLLM_MORI_DISPATCH_BLK_NO
      value: '128'
    - name: VLLM_MORI_DISPATCH_WARP_PER_BLK
      value: '16'
    - name: VLLM_MORI_COMBINE_BLK_NO
      value: '64'
    - name: VLLM_MORI_COMBINE_WARP_PER_BLK
      value: '8'
    - name: VLLM_IS_DECODE_WORKER
      value: 'decode'

prefill:
  replicas: 2

  <<: *common
  command:
    - /bin/bash
    - -lc

  args:
    - |
      vllm serve "/app/model/DeepSeek-R1" \
        --served-model-name deepseek-ai/DeepSeek-R1 \
        --port 8000 \
        --trust-remote-code \
        --tensor-parallel-size 1 \
        --data-parallel-size 8 \
        --enable-expert-parallel \
        --no-enable-prefix-caching \
        --no-enable-chunked-prefill \
        --enforce-eager \
        --max-model-len 8192 \
        --max-num-seqs 2048 \
        --kv-cache-dtype fp8_e4m3 \
        --quantization ds_fp8_per_token \
        --block-size 16 \
        --kv-transfer-config '{"kv_connector":"NixlConnector","kv_role":"kv_both"}' \
        --disable-uvicorn-access-log \
        --no-enable-log-requests \
        --disable-log-stats \
        --max-num-batched-token 64000 \
        --gpu-memory-utilization 0.92

  extraEnvVars:
    - name: VLLM_MOE_DP_CHUNK_SIZE
      value: '4096'
    - name: VLLM_V1_OUTPUT_PROC_CHUNK_SIZE
      value: '128'
    - name: VLLM_MORI_DISPATCH_BLK_NO
      value: '128'
    - name: VLLM_MORI_DISPATCH_WARP_PER_BLK
      value: '16'
    - name: VLLM_MORI_COMBINE_BLK_NO
      value: '64'
    - name: VLLM_MORI_COMBINE_WARP_PER_BLK
      value: '4'
    - name: VLLM_IS_DECODE_WORKER
      value: 'prefill'
```

</TabItem>
</Tabs>

Run the following commands to deploy and run the components.

**Istio gateway:**

```shell
kubectl apply -f gateway.yaml
kubectl get pod -n mif -l gateway.networking.k8s.io/gateway-name=mif
```

```shell Expected output
NAME                         READY   STATUS    RESTARTS   AGE
mif-istio-584474ddd9-rt9p9   1/1     Running   0          163m
```

**Heimdall scheduler:**

```shell
helm upgrade -i heimdall moreh/heimdall \
    --version v0.5.0 \
    -n mif \
    -f heimdall-values.yaml
kubectl get all -n mif -l app.kubernetes.io/instance=heimdall
```

```shell Expected output
NAME                            READY   STATUS    RESTARTS   AGE
pod/heimdall-5576d4f48b-bgn4c   1/1     Running   0          3d1h
```

**Odin inference service:**

```shell
helm upgrade -i inference-service moreh/inference-service \
    --version v0.6.1 \
    -n mif \
    -f inference-service-values.yaml
kubectl get all -n mif -l app.kubernetes.io/instance=inference-service
```

```shell Expected output
NAME                                             READY   STATUS    RESTARTS   AGE
pod/inference-service-decode-0-1                 1/1     Running   0          95s
pod/inference-service-decode-0-2                 1/1     Running   0          95s
pod/inference-service-decode-0-3                 1/1     Running   0          95s
pod/inference-service-decode-0-4                 1/1     Running   0          95s
pod/inference-service-decode-0-5                 1/1     Running   0          95s
pod/inference-service-decode-0-6                 1/1     Running   0          95s
pod/inference-service-decode-0-7                 1/1     Running   0          95s
pod/inference-service-decode-0-8                 1/1     Running   0          95s
pod/inference-service-decode-1-1                 1/1     Running   0          103s
pod/inference-service-decode-1-2                 1/1     Running   0          103s
pod/inference-service-decode-1-3                 1/1     Running   0          103s
pod/inference-service-decode-1-4                 1/1     Running   0          103s
pod/inference-service-decode-1-5                 1/1     Running   0          103s
pod/inference-service-decode-1-6                 1/1     Running   0          103s
pod/inference-service-decode-1-7                 1/1     Running   0          103s
pod/inference-service-decode-1-8                 1/1     Running   0          103s
pod/inference-service-decode-2-1                 1/1     Running   0          110s
pod/inference-service-decode-2-2                 1/1     Running   0          110s
pod/inference-service-decode-2-3                 1/1     Running   0          110s
pod/inference-service-decode-2-4                 1/1     Running   0          110s
pod/inference-service-decode-2-5                 1/1     Running   0          110s
pod/inference-service-decode-2-6                 1/1     Running   0          110s
pod/inference-service-decode-2-7                 1/1     Running   0          110s
pod/inference-service-decode-2-8                 1/1     Running   0          110s
pod/inference-service-prefill-648bfd7bd6-cthnv   1/1     Running   0          3m38s
pod/inference-service-prefill-648bfd7bd6-lz6km   1/1     Running   0          3m38s

NAME                                        READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/inference-service-prefill   2/2     2            2           3m38s

NAME                                                   DESIRED   CURRENT   READY   AGE
replicaset.apps/inference-service-prefill-648bfd7bd6   2         2         2       3m38s
```

---

## Benchmarking method

We follow a commonly used approach for measuring the computational performance of inference servers. Multiple concurrent users send requests at a specific request-per-second (RPS) rate, each with a fixed input sequence length and output sequence length. The concurrency and RPS are determined empirically as high as possible within the limits of GPU memory capacity and without allowing requests to accumulate in the request queue of vLLM instances. We measure the response times of these requests and compute output tokens per second, total tokens per second, time to first token, and inter-token latency (also known as time per output token).

We use the [vLLM bench serve](https://docs.vllm.ai/en/latest/cli/bench/serve/) tool to conduct experiments of this kind. However, this tool was originally designed to measure the performance of a single-GPU server, and several aspects of it are insufficient for evaluating the levels of throughput observed in our experiments &mdash; tens of thousands of tokens per second. Therefore, we implemented three additional features in the vLLM bench serve tool bundled with Moreh vLLM, to correctly measure performance in a distributed inference environment with very high throughput. See the modified version [here](https://github.com/moreh-dev/vllm/tree/main/vllm/benchmarks).

- `--warmup-time`, `--cooldown-time`: At the beginning of the experiment, before enough requests have accumulated, and near the end of the experiment, as computation winds down, the GPUs are not fully utilized. To reliably measure the maximum throughput achievable by the inference system, we enabled the tool to exclude requests from the initial (warm-up) and final (cool-down) phases from the performance measurement.
- `--max-connections-per-worker`: We made the response times of individual requests be recorded across multiple threads; otherwise, information for some requests may be lost.
- `--sharegpt-input-len`, `--sharegpt-output-len`, `--gutenberg-input-len`, `--gutenberg-output-len`: To accurately measure the effect of EP load balancing, we used substrings of meaningful text from a real dataset, cut to the desired input sequence length, as prompts rather than meaningless random strings.

In this benchmarking, we evaluate three different input/output sequence lengths (512/512, 1000/1000, and 2000/2000) and two different datasets ([ShareGPT](https://www.kaggle.com/datasets/roschildrui/sharegpt-v3-unfiltered-cleaned-split) and [Gutenberg](https://huggingface.co/datasets/manu/project_gutenberg)). To launch a new Moreh vLLM pod in a Kubernetes cluster, first create a `benchmarking-client.yaml` file as follows. **Please modify the following items to match your system.**

- **On lines 5, 15, 26 and 28, specify the name of the Kubernetes worker node on which the benchmarking pod will run.**
- **Store the `ShareGPT_V3_unfiltered_cleaned_split.json` file and the `project_gutenberg` directory on the host filesystem of that node, and specify their paths on lines 44 and 47.**

```yaml benchmarking-client.yaml {5,15,26,28,44,47}
apiVersion: v1
kind: Pod
metadata:
  annotations: {}
  name: <clientHostname>
  namespace: mif
spec:
  containers:
    - args:
        - infinity
      command:
        - sleep
      image: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm:vllm_251212
      imagePullPolicy: IfNotPresent
      name: <clientHostname>
      resources: {}
      volumeMounts:
        - name: sharegpt-dataset
          mountPath: '/app/dataset/ShareGPT_V3_unfiltered_cleaned_split.json'
        - name: gutenberg-dataset
          mountPath: '/app/dataset/project_gutenberg'
      securityContext:
        privileged: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: <clientHostname>
  nodeSelector:
    kubernetes.io/hostname: <clientHostname>
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
    - effect: NoSchedule
      key: amd.com/gpu
      operator: Exists
  volumes:
    - name: sharegpt-dataset
      hostPath:
        path: /path/to/ShareGPT_V3_unfiltered_cleaned_split.json
    - name: gutenberg-dataset
      hostPath:
        path: /path/to/project_gutenberg
```

Run the following command to start the pod.

```shell
kubectl -n mif apply -f benchmarking-client.yaml
```

Inside the pod, you can run `vllm bench serve` as follows. This is an example that uses an input sequence length of 512, an output sequence length of 512, and the ShareGPT dataset. **You may need to modify the host on line 6 depending on your Istio gateway address**.

```shell {6}
vllm bench serve \
  --backend vllm \
  --model "deepseek-ai/DeepSeek-R1" \
  --metric-percentiles "1,10,25,50,75,90" \
  --percentile-metrics "itl,tps,ttft" \
  --host "mif-istio.mif.svc.cluster.local" \
  --port 80 \
  --num-prompts 32400 \
  --max-concurrency 10800 \
  --request-rate 140 \
  --ignore-eos \
  --ready-check-timeout-sec 0 \
  --max-connections-per-worker 1296 \
  --warmup-time 120.0 \
  --cooldown-time 70.0 \
  --dataset-name sharegpt \
  --dataset-path /app/dataset/ShareGPT_V3_unfiltered_cleaned_split.json \
  --sharegpt-input-len 512 \
  --sharegpt-output-len 512
```

The followings are the actual commands used to run each experiment. You can click to view each command. For each experiment, the warm-up time and cool-down time were adjusted appropriately.

==- (512, 512, ShareGPT)

```shell
vllm bench serve \
  --backend vllm \
  --model "deepseek-ai/DeepSeek-R1" \
  --metric-percentiles "1,10,25,50,75,90" \
  --percentile-metrics "itl,tps,ttft" \
  --host "mif-istio.mif.svc.cluster.local" \
  --port 80 \
  --num-prompts 32400 \
  --max-concurrency 10800 \
  --request-rate 140 \
  --ignore-eos \
  --ready-check-timeout-sec 0 \
  --max-connections-per-worker 1296 \
  --warmup-time 120.0 \
  --cooldown-time 70.0 \
  --dataset-name sharegpt \
  --dataset-path /app/dataset/ShareGPT_V3_unfiltered_cleaned_split.json \
  --sharegpt-input-len 512 \
  --sharegpt-output-len 512
```

==- (512, 512, Gutenberg)

```shell
vllm bench serve \
  --backend vllm \
  --model "deepseek-ai/DeepSeek-R1" \
  --metric-percentiles "1,10,25,50,75,90" \
  --percentile-metrics "itl,tps,ttft" \
  --host "mif-istio.mif.svc.cluster.local" \
  --port 80 \
  --num-prompts 32400 \
  --max-concurrency 10800 \
  --request-rate 140 \
  --ignore-eos \
  --ready-check-timeout-sec 0 \
  --max-connections-per-worker 1296 \
  --warmup-time 130.0 \
  --cooldown-time 70.0 \
  --dataset-name gutenberg \
  --dataset-path /app/dataset/project_gutenberg \
  --gutenberg-input-len 512 \
  --gutenberg-output-len 512
```

==- (1000, 1000, ShareGPT)

```shell
vllm bench serve \
  --backend vllm \
  --model "deepseek-ai/DeepSeek-R1" \
  --metric-percentiles "1,10,25,50,75,90" \
  --percentile-metrics "itl,tps,ttft" \
  --host "mif-istio.mif.svc.cluster.local" \
  --port 80 \
  --num-prompts 32400 \
  --max-concurrency 10800 \
  --request-rate 80 \
  --ignore-eos \
  --ready-check-timeout-sec 0 \
  --max-connections-per-worker 1296 \
  --warmup-time 140.0 \
  --cooldown-time 110.0 \
  --dataset-name sharegpt \
  --dataset-path /app/dataset/ShareGPT_V3_unfiltered_cleaned_split.json \
  --sharegpt-input-len 1000 \
  --sharegpt-output-len 1000
```

==- (1000, 1000, Gutenberg)

```shell
vllm bench serve \
  --backend vllm \
  --model "deepseek-ai/DeepSeek-R1" \
  --metric-percentiles "10,25,50,75,90" \
  --percentile-metrics "itl,tps,ttft" \
  --host "mif-istio.mif.svc.cluster.local" \
  --port 80 \
  --num-prompts 32400 \
  --max-concurrency 10800 \
  --request-rate 80 \
  --ignore-eos \
  --ready-check-timeout-sec 0 \
  --max-connections-per-worker 1296 \
  --warmup-time 150.0 \
  --cooldown-time 120.0 \
  --dataset-name gutenberg \
  --dataset-path /app/dataset/project_gutenberg \
  --gutenberg-input-len 1000 \
  --gutenberg-output-len 1000
```

==- (2000, 2000, ShareGPT)

```shell
vllm bench serve \
  --backend vllm \
  --model "deepseek-ai/DeepSeek-R1" \
  --metric-percentiles "1,10,25,50,75,90" \
  --percentile-metrics "itl,tps,ttft" \
  --host "mif-istio.mif.svc.cluster.local" \
  --port 80 \
  --num-prompts 32400 \
  --max-concurrency 10800 \
  --request-rate 48 \
  --ignore-eos \
  --ready-check-timeout-sec 0 \
  --max-connections-per-worker 1296 \
  --warmup-time 250.0 \
  --cooldown-time 290.0 \
  --dataset-name sharegpt \
  --dataset-path /app/dataset/ShareGPT_V3_unfiltered_cleaned_split.json \
  --sharegpt-input-len 2000 \
  --sharegpt-output-len 2000
```

==- (2000, 2000, Gutenberg)

```shell
vllm bench serve \
  --backend vllm \
  --model "deepseek-ai/DeepSeek-R1" \
  --metric-percentiles "1,10,25,50,75,90" \
  --percentile-metrics "itl,tps,ttft" \
  --host "mif-istio.mif.svc.cluster.local" \
  --port 80 \
  --num-prompts 32400 \
  --max-concurrency 10800 \
  --request-rate 60 \
  --ignore-eos \
  --ready-check-timeout-sec 0 \
  --max-connections-per-worker 1296 \
  --warmup-time 260.0 \
  --cooldown-time 240.0 \
  --dataset-name gutenberg \
  --dataset-path /app/dataset/project_gutenberg \
  --gutenberg-input-len 2000 \
  --gutenberg-output-len 2000
```

===

---

## Experimental results

The results are as follows. As mentioned earlier, the concurrency and RPS values were determined empirically and may vary depending on the system scale (the number of GPU nodes). We achieved 50,892-66,194 output tokens/sec across various configurations, which corresponds to **17,000-22,000 tokens/sec per decode node**.

| Input sequence length | Output sequence length |  Dataset  | (Concurrency, RPS) | Output tokens/sec | Output tokens/sec per decode node | Total tokens/sec | Mean TTFT (ms) | Mean ITL (ms) |
| :-------------------: | :--------------------: | :-------: | :----------------: | ----------------: | --------------------------------: | ---------------: | -------------: | ------------: |
|          512          |          512           | ShareGPT  |    (10800, 140)    |         66,194.80 |                     **22,064.93** |        85,347.35 |       1,677.87 |        160.33 |
|          512          |          512           | Gutenberg |    (10800, 140)    |         64,695.10 |                     **21,565.03** |        79,432.24 |       1,774.90 |        164.32 |
|         1000          |          1000          | ShareGPT  |    (10800, 80)     |         61,828.90 |                     **20,609.63** |        94,103.87 |       1,802.87 |        172.16 |
|         1000          |          1000          | Gutenberg |    (10800, 80)     |         61,418.55 |                     **20,472.85** |        92,353.63 |       2,149.80 |        173.63 |
|         2000          |          2000          | ShareGPT  |    (10800, 48)     |         51,187.87 |                     **17,062.62** |        77,775.33 |       2,567.87 |        208.59 |
|         2000          |          2000          | Gutenberg |    (10800, 60)     |         50,892.65 |                     **16,964.22** |        76,739.86 |       5,586.34 |        208.76 |

Click to view raw benchmarking logs.

==- (512, 512, ShareGPT)

```
=============Serving Benchmark Result=============
Number of worker processes:              25
Successful requests:                     4333
Maximum request concurrency:             10800
Request rate configured (RPS):           140.00
Warm-up Time:                            120.0
Cool-down Time:                          70.0
Benchmark duration (s):                  115.83
Total input tokens:                      2218496
Total generated tokens:                  7667539
Output token throughput (tok/s):         66194.80
Total Token throughput (tok/s):          85347.35
---------------Time to First Token----------------
Mean TTFT (ms):                          1677.87
Median TTFT (ms):                        1720.09
P1 TTFT (ms):                            673.72
P10 TTFT (ms):                           944.53
P25 TTFT (ms):                           1177.09
P50 TTFT (ms):                           1720.09
P75 TTFT (ms):                           2117.51
P90 TTFT (ms):                           2387.32
---------------Inter-token Latency----------------
Mean ITL (ms):                           160.33
Median ITL (ms):                         158.98
P1 ITL (ms):                             105.28
P10 ITL (ms):                            139.59
P25 ITL (ms):                            151.42
P50 ITL (ms):                            158.98
P75 ITL (ms):                            169.65
P90 ITL (ms):                            184.02
==================================================
```

==- (512, 512, Gutenberg)

```
=============Serving Benchmark Result=============
Number of worker processes:              25
Successful requests:                     3186
Maximum request concurrency:             10800
Request rate configured (RPS):           140.00
Warm-up Time:                            130.0
Cool-down Time:                          70.0
Benchmark duration (s):                  110.69
Total input tokens:                      1631232
Total generated tokens:                  7161008
Output token throughput (tok/s):         64695.10
Total Token throughput (tok/s):          79432.24
---------------Time to First Token----------------
Mean TTFT (ms):                          1774.90
Median TTFT (ms):                        1795.76
P1 TTFT (ms):                            775.35
P10 TTFT (ms):                           877.82
P25 TTFT (ms):                           1124.76
P50 TTFT (ms):                           1795.76
P75 TTFT (ms):                           2296.75
P90 TTFT (ms):                           2685.10
---------------Inter-token Latency----------------
Mean ITL (ms):                           164.32
Median ITL (ms):                         162.24
P1 ITL (ms):                             106.99
P10 ITL (ms):                            142.68
P25 ITL (ms):                            154.22
P50 ITL (ms):                            162.24
P75 ITL (ms):                            174.62
P90 ITL (ms):                            189.84
==================================================
```

==- (1000, 1000, ShareGPT)

```
=============Serving Benchmark Result=============
Number of worker processes:              25
Successful requests:                     11856
Maximum request concurrency:             10800
Request rate configured (RPS):           80.00
Warm-up Time:                            140.0
Cool-down Time:                          110.0
Benchmark duration (s):                  367.34
Total input tokens:                      11856000
Total generated tokens:                  22712445
Output token throughput (tok/s):         61828.90
Total Token throughput (tok/s):          94103.87
---------------Time to First Token----------------
Mean TTFT (ms):                          1802.87
Median TTFT (ms):                        1411.86
P1 TTFT (ms):                            724.34
P10 TTFT (ms):                           987.73
P25 TTFT (ms):                           1059.29
P50 TTFT (ms):                           1411.86
P75 TTFT (ms):                           2221.47
P90 TTFT (ms):                           3434.72
---------------Inter-token Latency----------------
Mean ITL (ms):                           172.16
Median ITL (ms):                         169.69
P1 ITL (ms):                             120.22
P10 ITL (ms):                            157.36
P25 ITL (ms):                            164.97
P50 ITL (ms):                            169.69
P75 ITL (ms):                            177.89
P90 ITL (ms):                            192.85
==================================================
```

==- (1000, 1000, Gutenberg)

```
=============Serving Benchmark Result=============
Number of worker processes:              25
Successful requests:                     10931
Maximum request concurrency:             10800
Request rate configured (RPS):           80.00
Warm-up Time:                            150.0
Cool-down Time:                          120.0
Benchmark duration (s):                  353.35
Total input tokens:                      10931000
Total generated tokens:                  21702425
Output token throughput (tok/s):         61418.55
Total Token throughput (tok/s):          92353.63
---------------Time to First Token----------------
Mean TTFT (ms):                          2149.80
Median TTFT (ms):                        1910.21
P10 TTFT (ms):                           1040.06
P25 TTFT (ms):                           1374.56
P50 TTFT (ms):                           1910.21
P75 TTFT (ms):                           2759.64
P90 TTFT (ms):                           3502.56
---------------Inter-token Latency----------------
Mean ITL (ms):                           173.63
Median ITL (ms):                         171.10
P10 ITL (ms):                            155.99
P25 ITL (ms):                            165.99
P50 ITL (ms):                            171.10
P75 ITL (ms):                            180.85
P90 ITL (ms):                            195.95
==================================================
```

==- (2000, 2000, ShareGPT)

```
=============Serving Benchmark Result=============
Number of worker processes:              25
Successful requests:                     11906
Maximum request concurrency:             10800
Request rate configured (RPS):           48.00
Warm-up Time:                            300.0
Cool-down Time:                          230.0
Benchmark duration (s):                  895.61
Total input tokens:                      23812000
Total generated tokens:                  45844389
Output token throughput (tok/s):         51187.87
Total Token throughput (tok/s):          77775.33
---------------Time to First Token----------------
Mean TTFT (ms):                          2567.87
Median TTFT (ms):                        2538.34
P1 TTFT (ms):                            971.14
P10 TTFT (ms):                           1213.06
P25 TTFT (ms):                           1622.42
P50 TTFT (ms):                           2538.34
P75 TTFT (ms):                           3267.10
P90 TTFT (ms):                           4126.80
---------------Inter-token Latency----------------
Mean ITL (ms):                           208.59
Median ITL (ms):                         201.50
P1 ITL (ms):                             140.72
P10 ITL (ms):                            186.91
P25 ITL (ms):                            195.65
P50 ITL (ms):                            201.50
P75 ITL (ms):                            217.87
P90 ITL (ms):                            243.87
==================================================
```

==- (2000, 2000, Gutenberg)

```
=============Serving Benchmark Result=============
Number of worker processes:              25
Successful requests:                     12254
Maximum request concurrency:             10800
Request rate configured (RPS):           60.00
Warm-up Time:                            260.0
Cool-down Time:                          240.0
Benchmark duration (s):                  948.19
Total input tokens:                      24508000
Total generated tokens:                  48255768
Output token throughput (tok/s):         50892.65
Total Token throughput (tok/s):          76739.86
---------------Time to First Token----------------
Mean TTFT (ms):                          5586.34
Median TTFT (ms):                        5313.19
P1 TTFT (ms):                            1017.56
P10 TTFT (ms):                           1745.51
P25 TTFT (ms):                           2823.02
P50 TTFT (ms):                           5313.19
P75 TTFT (ms):                           7612.50
P90 TTFT (ms):                           10096.06
---------------Inter-token Latency----------------
Mean ITL (ms):                           208.76
Median ITL (ms):                         201.13
P1 ITL (ms):                             139.90
P10 ITL (ms):                            187.16
P25 ITL (ms):                            195.34
P50 ITL (ms):                            201.13
P75 ITL (ms):                            214.94
P90 ITL (ms):                            245.98
=================================================
```

===

The followings are some publicly available performance numbers for comparison.

- The SGLang team reported that, on a cluster of 12x H100 nodes (96x GPUs) &mdash; with 3 nodes used for prefill and 9 nodes for decode &mdash; they achieved a throughput of **22,300 output tokens/sec** per decode node under a configuration with an input sequence length of 2,000 and an output sequence length of 100. Note that this number does not represent end-to-end performance with actual PD disaggregation applied; rather, it measures partial performance with decoding-only execution. ([Link](https://lmsys.org/blog/2025-05-05-large-scale-ep/))
- DeepSeek reported achieving **14,800 tokens/sec** per H800 decode node by applying PD disaggregation and expert parallelism. ([Link](https://github.com/deepseek-ai/open-infra-index/blob/main/202502OpenSourceWeek/day_6_one_more_thing_deepseekV3R1_inference_system_overview.md))
- AMD reported achieving **up to 14,300 output tokens/sec** per MI300X decode node. This result was also measured under decoding-only execution. ([Link](https://rocm.blogs.amd.com/software-tools-optimization/wide-ep-deepseek/README.html)).

In real production deployments, an appropriate trade-off between throughput and latency (inter-token latency and time to first token) must be chosen according to the service-level objectives (SLOs). As shorter latency targets are pursued, achievable throughput inevitably decreases. Nevertheless, meausring and comparing the maximum achievable throughput before applying SLO constraints is an important step in evaluating infrastructure efficiency. Our next benchmarking will examine how throughput varies across different ITL targets.

---

## Appendix

### Experimental results for ISL=2,000 and OSL=100

An input sequence length of 2,000 and an output sequence length of 100 were first used by the SGLang team for their PD+EP performance evaluation. Since then, this configuration has been widely adopted to evaluate PD+EP performance of DeepSeek R1.

First, please note that this configuration was proposed to measure prefill and decode throughput separately. Under the assumption that the input length is always 20x longer, a real inference system would require ~10x more prefill instances than decode instances. (In practice, real usage patterns differ from this assumption, and the number of decode instances typically exceeds that of prefill instances.) In small clusters, prefill inevitably becomes the overall performance bottleneck, making it impossible to accurately measuring the output tokens/sec that the GPU servers can actually deliver.

Despite this, by enabling prefix caching and having input sequences share a fixed set of prompts, we can design a scenario in which the prefill workload is significantly reduced and measure the resulting output tokens/sec. As a result, we achieved **~18,000 tokens/sec per decode node**.

| Input sequence length | Output sequence length |  Dataset  | (Concurrency, RPS) | Output tokens/sec | Output tokens/sec per decode node | Mean ITL (ms) |
| :-------------------: | :--------------------: | :-------: | :----------------: | ----------------: | --------------------------------: | ------------: |
|         2000          |          100           | Gutenberg |   (10800, 1500)    |         53,776.62 |                     **17,925.54** |        191.71 |

Click to view the raw benchmarking log.

==- (2000, 100, Gutenberg)

```
'=============Serving Benchmark Result=============
Number of worker processes:              30
Successful requests:                     185577
Maximum request concurrency:             10800
Request rate configured (RPS):           1500.00
Warm-up Time:                            30.0
Cool-down Time:                          20.0
Benchmark duration (s):                  367.68
Total input tokens:                      371154000
Total generated tokens:                  19772555
Output token throughput (tok/s):         53776.62
Total Token throughput (tok/s):          1063226.66
---------------Time to First Token----------------
Mean TTFT (ms):                          1079.36
Median TTFT (ms):                        979.51
P10 TTFT (ms):                           832.05
P25 TTFT (ms):                           897.71
P50 TTFT (ms):                           979.51
P75 TTFT (ms):                           1079.35
P90 TTFT (ms):                           1223.28
---------------Inter-token Latency----------------
Mean ITL (ms):                           191.71
Median ITL (ms):                         181.35
P10 ITL (ms):                            166.87
P25 ITL (ms):                            175.00
P50 ITL (ms):                            181.35
P75 ITL (ms):                            191.88
P90 ITL (ms):                            259.46
==================================================
```

===

We have also measured the performance of a decoding-only execution under the same configuration (ISL=2,000, OSL=100) and reported the results in a [technical report](https://moreh.io/technical-report/21k-output-tokens-per-second-deepseek-inference-on-amd-instinct-mi300x-gpus-with-expert-parallelism-251113/). The maximum throughput achieved in this setting was 21,224 tokens/sec per decode node. This indicates that, in an end-to-end environment, MoAI Inference Framework is able to achieve **~85% of the peak decode performance**.
