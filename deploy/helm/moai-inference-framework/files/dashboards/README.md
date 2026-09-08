# MIF dashboards

The chart packages each JSON file in this directory as a Grafana dashboard ConfigMap.

| File | Dashboard UID | Scope |
| --- | --- | --- |
| `mif-scheduler.json` | `mif` | GPU utilization and current vLLM engine metrics |
| `mif-aigateway.json` | `mif-aigateway` | AIGateway request/tokenizer metrics and sidecar endpoint metrics |

The historical `mif-scheduler.json` filename and `mif` UID are retained to preserve provisioning identity and existing links. The engine dashboard uses `vllm:*`, not the legacy `llm_*` or `inference_extension_*` schema.

Both dashboards use a Prometheus data-source variable. Select the appropriate source in Grafana; do not commit an environment-specific data-source UID or database dashboard ID. Clear saved variable options and environment-specific selections when exporting a tested dashboard.

## Filters and metric meaning

- On MIF, GPU node selectors affect GPU panels. Namespace, gateway, model, role, and pod selectors affect engine panels.
- On MIF — AIGateway, model selects request/tokenizer metrics. Role and endpoint select endpoint metrics; a model selection does not restrict endpoint panels.
- Role filtering requires `kube_pod_labels` with `mif.moreh.io/role` allowlisted by kube-state-metrics. Engine and gateway metrics do not carry this role label directly. `All` includes pods without a role label; prefill/decode selections use their Kubernetes labels.
- Engine completion/token counters and engine latency are not client request throughput or gateway latency. In disaggregated serving, prefill and decode may count stages of the same client request. Select a role when comparing engine measurements.
- Histograms without observations in the rate interval produce no measured percentile. Do not replace these values with zero. Prefill engines may not generate inter-token observations; idle models may have no latency observations. Check request rates and expand the time range.
- Gateway endpoint counts deduplicate replicas. An endpoint counts as healthy only if every reporting gateway replica considers it healthy. Last-observation panels retain each replica's observation explicitly.
- Dashboard variables and panel queries are not access-control boundaries. External viewers need a restricted data source/backend if they must not query other namespaces or node-level information.

## Validation and provisioning

Validate the default selection, prefill/decode selections, a single model/pod, and an idle model against real Prometheus data. Check that data-source references resolve and endpoint counts do not grow with gateway replicas. Run `helm lint` and render `templates/grafana/dashboard-configmap.yaml`; parse the rendered JSON and verify unique dashboard UIDs.

Retire old ConfigMaps when migrating filenames: both `mif.json` and `mif-scheduler.json` with UID `mif` must not be supplied to the same Grafana organization. Keep one provisioning source per UID. Independently copied dashboards in another organization are not updated by the source ConfigMap.
