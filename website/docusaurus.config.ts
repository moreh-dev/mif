import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Moreh',
  tagline: 'MoAI Inference Framework documentation',
  url: 'https://docs.moreh.io',
  baseUrl: '/',
  favicon: 'moreh-icon.png',
  organizationName: 'moreh-dev',
  projectName: 'mif',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },
  staticDirectories: ['static', '../docs.moreh.io/docs/static'],
  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: require.resolve('./sidebars'),
          routeBasePath: '/',
        },
        blog: false,
        theme: {
          customCss: [],
        },
      } satisfies Preset.Options,
    ],
  ],
  themeConfig: {
    navbar: {
      title: 'Moreh',
      logo: {
        alt: 'Moreh logo',
        src: 'moreh-logo.svg',
      },
      items: [
        {
          type: 'doc',
          docId: 'home',
          position: 'left',
          label: 'Docs',
        },
        {
          href: 'https://moreh.io/',
          label: 'Website',
          position: 'right',
        },
        {
          href: 'https://github.com/moreh-dev/mif',
          label: 'GitHub',
          position: 'right',
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

