import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const includeCurrentVersion =
  process.env.DOCS_INCLUDE_CURRENT_VERSION !== 'false';

// Required by Docusaurus; use env for preview (GitHub Pages) vs release (custom domain).
// See: https://docusaurus.io/docs/deployment#configuration
const siteUrl = process.env.DOCS_SITE_URL ?? 'https://docs.moreh.io';
const baseUrl = process.env.DOCS_BASE_URL ?? '/';

const config: Config = {
  title: 'Moreh',
  tagline: 'MoAI Inference Framework documentation',
  url: siteUrl,
  baseUrl,
  favicon: '/moreh-icon.png',
  organizationName: 'moreh-dev',
  projectName: 'mif',
  staticDirectories: ['static'],
  onBrokenLinks: 'warn',
  plugins: [
    [
      '@cmfcmf/docusaurus-search-local',
      {
        indexBlog: false,
        language: 'en',
      },
    ],
  ],
  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: require.resolve('./sidebars'),
          routeBasePath: '/',
          includeCurrentVersion,
          ...(includeCurrentVersion && {
            versions: {
              current: {
                label: 'Dev 🚧',
                path: 'dev',
              },
            },
          }),
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
