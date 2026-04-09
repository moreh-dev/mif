import React from "react";
import Link from "@docusaurus/Link";
import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";

/* ─── data ───────────────────────────────────────────────────────── */

const coreCapabilities = [
  {
    label: "01",
    title: "SLO-Driven Optimization",
    description:
      "Specify latency constraints and let the framework automatically determine the optimal parallelization strategy and resource allocation to maximize throughput per dollar.",
  },
  {
    label: "02",
    title: "Prefill-Decode Disaggregation",
    description:
      "Separates prefill and decode phases across different GPU pools — including across heterogeneous GPU types — to optimize resource utilization for each workload characteristic.",
  },
  {
    label: "03",
    title: "Prefix Cache-Aware Routing",
    description:
      "Routes requests to instances with pre-cached prefix computations, reducing TTFT by up to 20x and achieving 2.2x throughput with just 40% of the servers.",
  },
  {
    label: "04",
    title: "Request Length-Based Routing",
    description:
      "Classifies incoming requests by expected length and routes them to GPU pools optimized for each workload profile — short prompts to latency-tuned instances, long contexts to throughput-tuned ones.",
  },
  {
    label: "05",
    title: "Auto Scaling",
    description:
      "Automatically scales inference capacity up and down based on traffic patterns, ensuring optimal resource utilization and cost efficiency.",
  },
];

const models = [
  { name: "DeepSeek", icon: "/assets/icon-hf-deepseek-ai.webp" },
  { name: "Llama", icon: "/assets/icon-hf-meta-llama.webp" },
  { name: "Qwen", icon: "/assets/icon-hf-qwen.webp" },
  { name: "Mistral", icon: "/assets/icon-hf-mistralai.webp" },
  { name: "Gemma", icon: "/assets/icon-hf-google.webp" },
  { name: "GLM", icon: "/assets/icon-hf-zai-org.webp" },
  { name: "Kimi", icon: "/assets/icon-hf-moonshotai.webp" },
  { name: "Step", icon: "/assets/icon-hf-stepfun-ai.webp" },
];

const chipGroups = [
  { vendor: "NVIDIA", chips: ["B300", "B200", "H200", "H100", "H20", "A100"] },
  {
    vendor: "AMD",
    chips: ["MI355X", "MI325X", "MI308X", "MI300X", "MI250X", "MI250"],
  },
  { vendor: "Tenstorrent", chips: ["Blackhole", "Wormhole"] },
];

const networkTypes = ["RoCE", "InfiniBand"];

const k8sTags = [
  "Kubernetes Native",
  "Gateway API Inference Extension",
  "Istio Compatible",
  "Helm Charts",
  "NFD Integration",
  "RoCE Networking",
];

/* ─── styles ─────────────────────────────────────────────────────── */

const sectionStyle: React.CSSProperties = {
  padding: "4rem 2rem",
};

const containerStyle: React.CSSProperties = {
  maxWidth: 1200,
  margin: "0 auto",
};

const labelStyle: React.CSSProperties = {
  color: "var(--ifm-color-primary)",
  fontWeight: 600,
  fontSize: "0.85rem",
  textTransform: "uppercase",
  letterSpacing: "0.05em",
  marginBottom: "0.5rem",
};

const tagStyle: React.CSSProperties = {
  padding: "0.5rem 1rem",
  fontSize: "0.875rem",
  borderRadius: 4,
  border: "1px solid var(--ifm-color-emphasis-300)",
  background: "var(--ifm-background-surface-color)",
};

/* ─── sections ───────────────────────────────────────────────────── */

function Hero() {
  return (
    <header style={{ ...sectionStyle, textAlign: "center" }}>
      <div style={containerStyle}>
        <p style={{ ...labelStyle, fontSize: "2.5rem" }}>MoAI Inference Framework</p>
        <h1 style={{ fontSize: "2.5rem", marginBottom: "1rem" }}>
          Automating distributed inference at data center scale
        </h1>
        <p
          style={{
            fontSize: "1.15rem",
            maxWidth: 760,
            margin: "0 auto 2rem",
            color: "var(--ifm-color-emphasis-700)",
          }}
        >
          Serve large models across every GPU you have — regardless of vendor,
          generation, or architecture — through a single API endpoint. MoAI
          Inference Framework automatically allocates resources, routes requests,
          and scales capacity so your cluster delivers maximum throughput at the
          lowest latency.
        </p>
        <div style={{ display: "flex", gap: "1rem", justifyContent: "center" }}>
          <Link className="button button--primary button--lg" to="/docs/getting-started/quickstart">
            Get started
          </Link>
        </div>
      </div>
    </header>
  );
}

function KeyDifferentiator() {
  return (
    <section className="hero-contrast-section" style={sectionStyle}>
      <div style={containerStyle}>
        <div className="landing-grid landing-grid--3-2">
          <div>
            <p style={{ ...labelStyle, color: "var(--ifm-color-primary)" }}>
              Key Differentiator
            </p>
            <h2 className="hero-contrast-title" style={{ fontSize: "2rem", marginBottom: "0.75rem" }}>
              One Cluster, Every GPU
            </h2>
            <p className="hero-contrast-text" style={{ marginBottom: "1.5rem" }}>
              Most inference stacks lock you into a single vendor. MoAI
              Inference Framework breaks that constraint — split prefill and
              decode across chips from different vendors, squeeze remaining value
              out of legacy GPUs, or add non-GPU accelerators into the same
              cluster. Each device runs what it's best at.
            </p>
            <div style={{ display: "flex", gap: "2rem" }}>
              <div>
                <p
                  style={{
                    fontSize: "1.75rem",
                    fontWeight: 700,
                    color: "var(--ifm-color-primary)",
                    margin: 0,
                  }}
                >
                  1.7x
                </p>
                <p className="hero-contrast-stat-text" style={{ fontSize: "0.85rem", margin: 0 }}>
                  throughput with cross-vendor
                  <br />
                  PD disaggregation
                </p>
              </div>
              <div>
                <p
                  style={{
                    fontSize: "1.75rem",
                    fontWeight: 700,
                    color: "var(--ifm-color-primary)",
                    margin: 0,
                  }}
                >
                  0
                </p>
                <p className="hero-contrast-stat-text" style={{ fontSize: "0.85rem", margin: 0 }}>
                  overhead in mixed-vendor
                  <br />
                  unified routing
                </p>
              </div>
            </div>
          </div>

          {/* Diagram */}
          <div
            className="hero-contrast-diagram"
            style={{
              borderRadius: 4,
              padding: "2rem",
              textAlign: "center",
            }}
          >
            <div
              style={{
                background: "var(--ifm-color-primary)",
                borderRadius: 4,
                padding: "0.75rem 1rem",
                marginBottom: "0.75rem",
                color: "#fff",
                fontWeight: 500,
                fontSize: "0.9rem",
              }}
            >
              Unified API Endpoint
            </div>
            <div style={{ display: "flex", justifyContent: "center", marginBottom: "0.75rem" }}>
              <div className="hero-contrast-diagram-line" style={{ width: 1, height: 24 }} />
            </div>
            <div
              className="hero-contrast-diagram-box"
              style={{
                borderRadius: 4,
                padding: "0.625rem 1rem",
                marginBottom: "0.75rem",
                fontSize: "0.85rem",
              }}
            >
              Performance Gateway
            </div>
            <div style={{ display: "flex", gap: 8, marginBottom: "0.75rem" }}>
              {[0, 1, 2].map((i) => (
                <div key={i} style={{ flex: 1, display: "flex", justifyContent: "center" }}>
                  <div className="hero-contrast-diagram-line" style={{ width: 1, height: 20 }} />
                </div>
              ))}
            </div>
            <div style={{ display: "flex", gap: 8, marginBottom: "0.75rem" }}>
              {[
                { label: "NVIDIA", color: "rgb(74,222,128)" },
                { label: "AMD", color: "rgb(248,113,113)" },
                { label: "Tenstorrent", color: "rgb(192,132,252)" },
              ].map((g) => (
                <div
                  key={g.label}
                  className="hero-contrast-gpu"
                  style={{
                    flex: 1,
                    color: g.color,
                    borderRadius: 4,
                    padding: "0.625rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  {g.label}
                </div>
              ))}
            </div>
            <div style={{ display: "flex", gap: 8, marginBottom: "0.75rem" }}>
              {[0, 1, 2].map((i) => (
                <div key={i} style={{ flex: 1, display: "flex", justifyContent: "center" }}>
                  <div className="hero-contrast-diagram-line" style={{ width: 1, height: 20 }} />
                </div>
              ))}
            </div>
            <div
              className="hero-contrast-diagram-box"
              style={{
                borderRadius: 4,
                padding: "0.625rem 1rem",
                fontSize: "0.85rem",
              }}
            >
              Cross-Vendor Software Fabric
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function CoreCapabilities() {
  return (
    <section
      style={{
        ...sectionStyle,
        borderTop: "1px solid var(--ifm-color-emphasis-200)",
      }}
    >
      <div style={containerStyle}>
        <p style={labelStyle}>Core Capabilities</p>
        <h2 style={{ fontSize: "2rem", marginBottom: "0.5rem" }}>
          Automatic Optimization
        </h2>
        <p
          style={{
            color: "var(--ifm-color-emphasis-700)",
            maxWidth: 720,
            marginBottom: "2.5rem",
          }}
        >
          Efficient distributed inference requires combining multiple techniques,
          allocating GPU resources optimally, and scheduling requests
          intelligently. MoAI Inference Framework automates all of this based on
          your defined SLOs and real-time traffic patterns.
        </p>
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
            gap: "1.5rem",
          }}
        >
          {coreCapabilities.map((f) => (
            <div
              key={f.title}
              style={{
                padding: "1.5rem",
                borderRadius: 4,
                border: "1px solid var(--ifm-color-emphasis-300)",
                background: "var(--ifm-background-surface-color)",
              }}
            >
              <p
                style={{
                  color: "var(--ifm-color-primary)",
                  fontWeight: 500,
                  fontSize: "0.85rem",
                  marginBottom: "0.5rem",
                }}
              >
                {f.label}
              </p>
              <h3 style={{ fontSize: "1.1rem", marginBottom: "0.75rem" }}>
                {f.title}
              </h3>
              <p style={{ color: "var(--ifm-color-emphasis-700)", fontSize: "0.9rem", margin: 0 }}>
                {f.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function KubernetesNative() {
  return (
    <section className="hero-contrast-section" style={sectionStyle}>
      <div style={containerStyle}>
        <div className="landing-grid landing-grid--1-1">
          <div>
            <p style={{ ...labelStyle, color: "var(--ifm-color-primary)" }}>
              Architecture
            </p>
            <h2 className="hero-contrast-title" style={{ fontSize: "2rem", marginBottom: "1rem" }}>
              Kubernetes Native
            </h2>
            <p className="hero-contrast-text">
              MoAI Inference Framework runs as a set of Kubernetes-native
              controllers — no sidecar daemons, no proprietary control plane.
              Deploy with Helm, expose through any Gateway API Inference
              Extension-compatible controller including Istio, and let NFD
              auto-discover heterogeneous accelerators across your fleet.
            </p>
          </div>
          <div style={{ display: "flex", flexWrap: "wrap", gap: "0.75rem" }}>
            {k8sTags.map((tag) => (
              <span key={tag} className="hero-contrast-diagram-box" style={{
                padding: "0.5rem 1rem",
                fontSize: "0.875rem",
                borderRadius: 4,
              }}>
                {tag}
              </span>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function SupportedModelsAndHardware() {
  return (
    <section
      style={{
        ...sectionStyle,
        borderTop: "1px solid var(--ifm-color-emphasis-200)",
      }}
    >
      <div style={containerStyle}>
        <div className="landing-grid landing-grid--1-1">
          {/* Models */}
          <div>
            <h2 style={{ fontSize: "2rem", marginBottom: "0.5rem" }}>
              Supported Models
            </h2>
            <p style={{ color: "var(--ifm-color-emphasis-700)", marginBottom: "1.5rem" }}>
              Works with any model supported by its underlying serving engines
              (Moreh vLLM, vLLM, SGLang, and others), including most open-source LLMs:
            </p>
            <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
              {models.map((model) => (
                <span key={model.name} style={{ ...tagStyle, display: "inline-flex", alignItems: "center", gap: "0.5rem" }}>
                  <img src={model.icon} alt={model.name} width={20} height={20} style={{ borderRadius: 4 }} />
                  {model.name}
                </span>
              ))}
              <span style={{ padding: "0.5rem 1rem", fontSize: "0.875rem", color: "var(--ifm-color-emphasis-500)" }}>
                and more
              </span>
            </div>
          </div>

          {/* Hardware */}
          <div>
            <h2 style={{ fontSize: "2rem", marginBottom: "2rem" }}>
              Supported Hardware
            </h2>
            <div style={{ marginBottom: "2rem" }}>
              <p style={{ fontWeight: 500, marginBottom: "0.75rem" }}>Accelerators</p>
              {chipGroups.map((group) => (
                <div
                  key={group.vendor}
                  style={{
                    display: "flex",
                    alignItems: "flex-start",
                    gap: "0.75rem",
                    marginBottom: "0.5rem",
                  }}
                >
                  <span
                    style={{
                      fontSize: "0.875rem",
                      color: "var(--ifm-color-emphasis-600)",
                      minWidth: 90,
                      paddingTop: "0.375rem",
                    }}
                  >
                    {group.vendor}
                  </span>
                  <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
                    {group.chips.map((chip) => (
                      <span key={chip} style={tagStyle}>
                        {chip}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
            <div>
              <p style={{ fontWeight: 500, marginBottom: "0.75rem" }}>Networking</p>
              <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
                {networkTypes.map((net) => (
                  <span key={net} style={tagStyle}>
                    {net}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ─── page ───────────────────────────────────────────────────────── */

export default function Home(): React.JSX.Element {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout title={siteConfig.title} description={siteConfig.tagline}>
      <main>
        <Hero />
        <KeyDifferentiator />
        <CoreCapabilities />
        <KubernetesNative />
        <SupportedModelsAndHardware />
      </main>
    </Layout>
  );
}
