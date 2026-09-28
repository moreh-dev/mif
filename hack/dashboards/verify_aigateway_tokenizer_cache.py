#!/usr/bin/env python3
"""Run the AIGateway dashboard's tokenizer cache panel queries against a local
Prometheus fed by fake gateways, and check the values each panel reports.

The fake gateways model a rolling rename of the lookup counters:

- model A, new pod: exports lookup_hits_total / lookup_misses_total and
  encoded_tokens_total.
- model A, old pod: exports only hits_total / misses_total.
- model B, old pod only.

Usage:
    python3 hack/dashboards/verify_aigateway_tokenizer_cache.py --prometheus /path/to/prometheus

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

# Per-second rates each fake pod exports, keyed by metric suffix.
PODS = {
    "new-a": (
        "A",
        {
            "lookup_hits_total": 10,
            "lookup_misses_total": 10,
            "hit_prefix_tokens_total": 20,
            "encoded_tokens_total": 1000,
        },
    ),
    "old-a": (
        "A",
        {"hits_total": 30, "misses_total": 10, "hit_prefix_tokens_total": 1500},
    ),
    "old-b": (
        "B",
        {"hits_total": 5, "misses_total": 5, "hit_prefix_tokens_total": 50},
    ),
}

# Expected value per (panel id, target refId, model). A missing model means
# the query must return no series for it.
EXPECTED = {
    # Only the new pod's reused tokens count, over its own encoded tokens.
    (77, "A"): {"A": 20 / 1000},
    # Lookups from both pods of model A, whichever name each exports.
    (78, "A"): {"A": 10 + 30, "B": 5},
    (78, "B"): {"A": 10 + 10, "B": 5},
    # Reused tokens per hit across both pods of model A.
    (79, "A"): {"A": (20 + 1500) / (10 + 30), "B": 50 / 5},
}

RATE_WINDOW = "20s"


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def serve_pod(port, model, rates, started):
    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            elapsed = time.time() - started
            body = "".join(
                f'aigateway_tokenizer_cache_{name}{{{LABELS},model="{model}"}} {rate * elapsed}\n'
                for name, rate in rates.items()
            )
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; version=0.0.4")
            self.end_headers()
            self.wfile.write(body.encode())

        def log_message(self, *args):
            pass

    server = http.server.HTTPServer(("127.0.0.1", port), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()


def query(base, expr):
    url = f"{base}/api/v1/query?" + urllib.parse.urlencode({"query": expr})
    with urllib.request.urlopen(url) as resp:
        body = json.load(resp)
    if body["status"] != "success":
        raise RuntimeError(f"query failed: {body}")
    return {r["metric"].get("model"): float(r["value"][1]) for r in body["data"]["result"]}


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
    for model, rates in PODS.values():
        port = free_port()
        serve_pod(port, model, rates, started)
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
                        "min(count_over_time(aigateway_tokenizer_cache_hit_prefix_tokens_total[1m]))",
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
                    math.isclose(got[m], want[m], rel_tol=1e-3) for m in want
                )
                failed |= not ok
                print(f"{'ok  ' if ok else 'FAIL'} panel {key[0]} {key[1]} {title}: got {got}, want {want}")
            sys.exit(1 if failed else 0)
        finally:
            prom.terminate()
            prom.wait()


if __name__ == "__main__":
    main()
