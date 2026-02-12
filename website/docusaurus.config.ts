import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const organizationName = process.env.ORGANIZATION_NAME;
const projectName = process.env.PROJECT_NAME;
const deploymentBranch = process.env.DEPLOYMENT_BRANCH;

const siteUrl = organizationName
  ? `https://${organizationName}.github.io`
  : undefined;
const baseUrl = projectName ? `/${projectName}/` : undefined;

const config: Config = {
  title: 'Moreh',
  tagline: 'MoAI Inference Framework documentation',
  url: siteUrl,
  baseUrl: baseUrl,
  organizationName: organizationName,
  projectName: projectName,
  deploymentBranch: deploymentBranch,
  trailingSlash: true,
  favicon: '/moreh-icon.png',
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
