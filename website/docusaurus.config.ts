import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Moreh',
  tagline: 'MoAI Inference Framework documentation',
  url: 'https://docs.moreh.io',
  baseUrl: '/',
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
        },
        theme: {
          customCss: require.resolve('./css/custom.css'),
        },
        blog: false,
      } satisfies Preset.Options,
    ],
  ],
  themeConfig: {
    navbar: {
      title: '',
      logo: {
        alt: 'Moreh logo',
        src: '/moreh-logo.svg',
        srcDark: '/moreh-logo-white.svg',
      },
      items: [
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
