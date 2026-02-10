---
id: home
title: Home
---

**MoAI Inference Framework** is a distributed inference framework that optimizes LLM inference at data center scale.

* **Support for diverse accelerators**: Supports AMD GPUs and Tenstorrent AI accelerators in addition to NVIDIA GPUs, enabling broader chip options for AI data centers. The entire software stack from GPU kernels and libraries to model implementation and distributed inference is highly optimized for such non-NVIDIA accelerators to deliver performance comparable to &mdash; or even surpassing &mdash; that of NVIDIA.
* **Model disaggregation and parallelization**: Applies model disaggregation techniques such as prefill-decode disaggregation and expert parallelism to maximize overall throughput of the entire cluster.
* **Optimal routing and scheduling**: Distributes incoming requests to the most suitable inference instances by considering various factors such as prefix cache locality and performance characteristics, resulting in better latency and throughput compared to using a simple load balancer.
* **Auto scaling**: Dynamically adjusts both the total number of GPUs and the number of GPUs assigned to each disaggregated model, depending on the amount and pattern of incoming requests. This ensures efficient resource utilization at data center scale.
* **Heterogeneous accelerator utilization**: Distributes different workloads (e.g., prefill and decode) across different types of accelerators to improve the overall efficiency of the system. For example, it can mix older and newer GPUs, NVIDIA and AMD GPUs, or even combine GPUs with CPX or Tenstorrent chips.
* **SLO-based automated distributed inference**: Automatically combines all the aforementioned techniques to maximize system throughput while satisfying defined service level objectives (SLOs).

## Materials

* [Distributed Inference on Heterogeneous Accelerators Including GPUs, Rubin CPX, and AI Accelerators](https://moreh.io/blog/distributed-inference-on-heterogeneous-accelerators-including-gpus-rubin-cpx-and-ai-accelerators-250923/) (blog article)
* [Moreh vLLM Performance Evaluation: DeepSeek V3/R1 671B on AMD Instinct MI300X GPUs](https://moreh.io/technical-report/moreh-vllm-performance-evaluation-deepseek-v3-r1-671b-on-amd-instinct-mi300x-gpus-250829/) (technical report)
* [Moreh vLLM Performance Evaluation: Llama 3.3 70B on AMD Instinct MI300X GPUs](https://moreh.io/technical-report/moreh-vllm-performance-evaluation-llama-3-3-70b-on-amd-instinct-mi300x-gpus-250830/) (technical report)
* [21K Output Tokens Per Second DeepSeek Inference on AMD Instinct MI300X GPUs with Expert Parallelism](https://moreh.io/technical-report/21k-output-tokens-per-second-deepseek-inference-on-amd-instinct-mi300x-gpus-with-expert-parallelism-251113/) (technical report)
* [Moreh-Tenstorrent AI Data Center Solution System Architecture](https://moreh.io/technical-report/moreh-tenstorrent-ai-data-center-solution-system-architecture-251118/) (technical report)
* [Optimizing Long-Context Prefill on Multiple (Older-Generation) GPU Nodes](https://moreh.io/blog/optimizing-long-context-prefill-on-multiple-older-generation-gpu-nodes-251226/) (blog article)

