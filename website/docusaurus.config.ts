import type {Config} from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";
import {themes} from "prism-react-renderer";
import versions from "./versions.json";

function getLastStableVersion(): string {
  const lastStable = versions.find((v) => !v.includes("-"));
  if (!lastStable) {
    throw new Error("No stable version found in versions.json");
  }
  return lastStable;
}

// Paths served by the Retype site that previously occupied docs.moreh.io.
// Emitted as static redirect pages because GitHub Pages has no server-side
// redirect facility.
const retypeRedirects = [
  {from: "/getting_started/overview", to: "/"},
  {
    from: "/getting_started/prerequisites",
    to: "/docs/getting-started/prerequisites",
  },
  {from: "/getting_started/quickstart", to: "/docs/getting-started/quickstart"},
  {
    from: "/getting_started/supported_devices",
    to: "/docs/reference/supported-devices",
  },
  {from: "/getting_started/logs", to: "/docs/operations/monitoring/logs"},
  {
    from: "/getting_started/monitoring",
    to: "/docs/operations/monitoring/metrics",
  },
  {from: "/features/preset", to: "/docs/features/preset"},
  {
    from: "/features/prefill_decode_disaggregation",
    to: "/docs/features/prefill-decode-disaggregation",
  },
  {
    from: "/features/prefix_cache_aware_routing",
    to: "/docs/features/prefix-cache-aware-routing",
  },
  {
    from: "/best_practices/container_image_caching_with_harbor",
    to: "/docs/operations/container-image-caching-with-harbor",
  },
  {
    from: "/best_practices/hf_model_management_with_pv",
    to: "/docs/operations/hf-model-management-with-pv",
  },
  {
    from: "/benchmarking/deepseek_r1_671b_on_amd_mi300x_gpus_maximum_throughput",
    to: "/blog/2025/11/11/deepseek-r1-671b-on-amd-mi300x-gpus-maximum-throughput",
  },
  {from: "/reference/heimdall_scheduler", to: "/docs/reference/heimdall/usage"},
  {
    from: "/reference/odin_inference_service",
    to: "/docs/reference/odin/api-reference",
  },
  {
    from: "/reference/odin_inference_service_template",
    to: "/docs/reference/odin/api-reference",
  },
];

const config: Config = {
  title: "Moreh",
  tagline: "MoAI Inference Framework documentation",
  url: "https://docs.moreh.io/",
  baseUrl: "/",
  trailingSlash: true,
  favicon: "/moreh-icon.png",
  staticDirectories: ["static"],
  onBrokenLinks: "throw",
  plugins: [
    [
      "@cmfcmf/docusaurus-search-local",
      {
        indexBlog: false,
        language: "en",
      },
    ],
    "docusaurus-plugin-image-zoom",
    [
      "@docusaurus/plugin-client-redirects",
      {
        redirects: retypeRedirects,
      },
    ],
  ],

  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: "warn",
    },
  },
  themes: ["@docusaurus/theme-mermaid"],

  presets: [
    [
      "classic",
      {
        docs: {
          sidebarPath: require.resolve("./sidebars"),
          lastVersion: getLastStableVersion(),
          versions: {
            current: {
              label: "Dev 🚧",
              path: "dev",
            },
          },
        },
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
        blog: {
          blogTitle: "Blog",
        },
      } satisfies Preset.Options,
    ],
  ],
  themeConfig: {
    colorMode: {
      defaultMode: "dark",
      respectPrefersColorScheme: true,
      disableSwitch: false,
    },
    navbar: {
      title: "",
      logo: {
        alt: "Moreh logo",
        src: "/moreh-logo.svg",
        srcDark: "/moreh-logo-white.svg",
      },
      items: [
        {
          to: "/docs/getting-started/quickstart",
          label: "Docs",
          position: "left",
        },
        {
          to: "blog",
          label: "Blog",
          position: "left",
        },
        {
          type: "docsVersionDropdown",
          position: "right",
        },
        {
          href: "https://moreh.io/",
          label: "Website",
          position: "right",
        },
        {
          href: "https://github.com/moreh-dev/mif",
          position: "right",
          className: "header-github-link",
          "aria-label": "GitHub repository",
        },
      ],
    },
    prism: {
      additionalLanguages: ["bash", "toml", "yaml", "promql"],
      theme: themes.nightOwlLight,
      darkTheme: themes.vsDark,
    },
    footer: {
      style: "dark",
      copyright:
        "© Copyright " +
        new Date().getFullYear() +
        " Moreh, Inc. All rights reserved.",
    },
    zoom: {
      selector: ".markdown img",
      background: {
        light: "rgb(255, 255, 255)",
        dark: "rgb(50, 50, 50)",
      },
    },
  },
};

export default config;
