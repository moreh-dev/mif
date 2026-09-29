#!/usr/bin/env python3
"""Run the AIGateway dashboard's request-observed KV cache hit panel queries
against a local Prometheus fed by fake gateways, and check the values each
panel reports.

The fake gateways export aigateway_prompt_tokens_observed_total and
aigateway_cached_tokens_reports_total as the gateway would after these
requests per second:

- model A, replica 1: one reported request with 64 of 100 prompt tokens
  cached.
- model A, replica 2: one reported request with 0 of 100 cached, and one
  request with no cached_tokens (absent).
- model B: two absent requests only, like a vLLM worker without
  --enable-prompt-tokens-details. The gateway never creates the observed
  token series for it.
- model C: one reported request with 0 of 50 cached, a real full miss.

Usage:
    python3 hack/dashboards/verify_aigateway_kv_cache_hit.py --prometheus /path/to/prometheus

The Prometheus binary comes from https://github.com/prometheus/prometheus/releases.
Exits non-zero when a panel query returns a value other than the expected one.
"""

import argparse
import http.server
import json
import math
import pathlib
import socket
import subprocess
import sys
import tempfile
import threading
import time
import urllib.parse
import urllib.request

DASHBOARD = (
    pathlib.Path(__file__).resolve().parents[2]
    / "deploy/helm/moai-inference-framework/files/dashboards/mif-aigateway.json"
)

LABELS = 'namespace="ns",aigateway="gw"'

# Per-second rates each fake gateway replica exports, as
# (metric, extra labels) -> rate.
PODS = {
    "a-1": {
        ('aigateway_prompt_tokens_observed_total', 'model="A",source="cache"'): 64,
        ('aigateway_prompt_tokens_observed_total', 'model="A",source="prefill"'): 36,
        ('aigateway_cached_tokens_reports_total', 'model="A",result="reported"'): 1,
    },
    "a-2": {
        ('aigateway_prompt_tokens_observed_total', 'model="A",source="cache"'): 0,
        ('aigateway_prompt_tokens_observed_total', 'model="A",source="prefill"'): 100,
        ('aigateway_cached_tokens_reports_total', 'model="A",result="reported"'): 1,
        ('aigateway_cached_tokens_reports_total', 'model="A",result="absent"'): 1,
    },
    "b": {
        ('aigateway_cached_tokens_reports_total', 'model="B",result="absent"'): 2,
    },
    "c": {
        ('aigateway_prompt_tokens_observed_total', 'model="C",source="cache"'): 0,
        ('aigateway_prompt_tokens_observed_total', 'model="C",source="prefill"'): 50,
        ('aigateway_cached_tokens_reports_total', 'model="C",result="reported"'): 1,
    },
}

# Expected value per (panel id, target refId), keyed by the series' labels
# other than model, joined with the model. A missing key means the query must
# return no series for it.
EXPECTED = {
    # Cached over engine prompt tokens, across both replicas of model A. Model
    # B reported nothing, so it has no ratio instead of a false 0%. Model C
    # reported a real 0.
    (84, "A"): {("A",): 64 / 200, ("C",): 0.0},
    # Model A: 2 of 3 requests reported. Model B reported none, which must
    # show as 0% and not disappear.
    (85, "A"): {("A",): 2 / 3, ("B",): 0.0, ("C",): 1.0},
    (86, "A"): {
        ("A", "cache"): 64,
        ("A", "prefill"): 136,
        ("C", "cache"): 0,
        ("C", "prefill"): 50,
    },
    (87, "A"): {
        ("A", "reported"): 2,
        ("A", "absent"): 1,
        ("B", "absent"): 2,
        ("C", "reported"): 1,
    },
}

RATE_WINDOW = "20s"


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def serve_pod(port, rates, started):
    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            elapsed = time.time() - started
            body = "".join(
                f"{name}{{{LABELS},{labels}}} {rate * elapsed}\n"
                for (name, labels), rate in rates.items()
            )
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; version=0.0.4")
            self.end_headers()
            self.wfile.write(body.encode())

        def log_message(self, *args):
            pass

    server = http.server.HTTPServer(("127.0.0.1", port), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()


def series_key(metric):
    extra = metric.get("source") or metric.get("result")
    return (metric.get("model"), extra) if extra else (metric.get("model"),)


def query(base, expr):
    url = f"{base}/api/v1/query?" + urllib.parse.urlencode({"query": expr})
    with urllib.request.urlopen(url) as resp:
        body = json.load(resp)
    if body["status"] != "success":
        raise RuntimeError(f"query failed: {body}")
    return {series_key(r["metric"]): float(r["value"][1]) for r in body["data"]["result"]}


def panel_exprs():
    dashboard = json.loads(DASHBOARD.read_text())
    panels = {p["id"]: p for p in dashboard["panels"]}
    for (panel_id, ref_id) in EXPECTED:
        target = next(t for t in panels[panel_id]["targets"] if t["refId"] == ref_id)
        expr = (
            target["expr"]
            .replace("$namespace", ".*")
            .replace("$aigateway", ".*")
            .replace("$model", ".*")
            .replace("$__rate_interval", RATE_WINDOW)
        )
        yield (panel_id, ref_id), panels[panel_id]["title"], expr


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--prometheus", default="prometheus", help="Prometheus binary")
    args = parser.parse_args()

    started = time.time()
    targets = []
    for rates in PODS.values():
        port = free_port()
        serve_pod(port, rates, started)
        targets.append(f"127.0.0.1:{port}")

    prom_port = free_port()
    base = f"http://127.0.0.1:{prom_port}"
    with tempfile.TemporaryDirectory() as tmp:
        config = pathlib.Path(tmp, "prometheus.yml")
        config.write_text(
            "global: {scrape_interval: 1s}\n"
            "scrape_configs:\n"
            "  - job_name: fake-aigateway\n"
            "    honor_labels: true\n"
            f"    static_configs: [{{targets: {json.dumps(targets)}}}]\n"
        )
        prom = subprocess.Popen(
            [
                args.prometheus,
                f"--config.file={config}",
                f"--web.listen-address=127.0.0.1:{prom_port}",
                f"--storage.tsdb.path={tmp}/data",
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        try:
            # Enough samples in every series for a full rate window.
            deadline = time.time() + 120
            while True:
                try:
                    counts = query(
                        base,
                        "min(count_over_time(aigateway_cached_tokens_reports_total[1m]))",
                    )
                    if counts and min(counts.values()) >= 25:
                        break
                except OSError:
                    pass
                if time.time() > deadline:
                    sys.exit("Prometheus collected no samples within 120s")
                time.sleep(1)

            failed = False
            for key, title, expr in panel_exprs():
                got = query(base, expr)
                want = EXPECTED[key]
                ok = got.keys() == want.keys() and all(
                    math.isclose(got[k], want[k], rel_tol=1e-3, abs_tol=1e-9) for k in want
                )
                failed |= not ok
                print(f"{'ok  ' if ok else 'FAIL'} panel {key[0]} {key[1]} {title}: got {got}, want {want}")
            sys.exit(1 if failed else 0)
        finally:
            prom.terminate()
            prom.wait()


if __name__ == "__main__":
    main()
