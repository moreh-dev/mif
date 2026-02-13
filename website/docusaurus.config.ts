import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import {themes} from 'prism-react-renderer';

const config: Config = {
  title: 'Moreh',
  tagline: 'MoAI Inference Framework documentation',
  url: 'https://test-docs.moreh.io/',
  baseUrl: '/',
  trailingSlash: true,
  favicon: '/moreh-icon.png',
  staticDirectories: ['static'],
  onBrokenLinks: 'throw',
  plugins: [
    [
      '@cmfcmf/docusaurus-search-local',
      {
        indexBlog: false,
        language: 'en',
      },
    ],
  ],

  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },
  themes: ['@docusaurus/theme-mermaid'],

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: require.resolve('./sidebars'),
          routeBasePath: '/',
          versions: {
            current: {
              label: 'Dev 🚧',
              path: 'dev',
            },
          },
        },
        theme: {
          customCss: require.resolve('./css/custom.css'),
        },
        blog: false,
      } satisfies Preset.Options,
    ],
  ],
  themeConfig: {
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: true,
      disableSwitch: false,
    },
    navbar: {
      title: '',
      logo: {
        alt: 'Moreh logo',
        src: '/moreh-logo.svg',
        srcDark: '/moreh-logo-white.svg',
      },
      items: [
        {
          type: 'docsVersionDropdown',
          position: 'right',
        },
        {
          href: 'https://moreh.io/',
          label: 'Website',
          position: 'right',
        },
        {
          href: 'https://github.com/moreh-dev/mif',
          position: 'right',
          className: 'header-github-link',
          'aria-label': 'GitHub repository',
        },
      ],
    },
    prism: {
      additionalLanguages: ['bash', 'toml', 'yaml'],
      theme: themes.nightOwlLight,
      darkTheme: themes.vsDark,
    },
    footer: {
      style: 'dark',
      copyright:
        '© Copyright ' +
        new Date().getFullYear() +
        ' Moreh, Inc. All rights reserved.',
    },
  },
};

export default config;
