// @ts-check

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Betar',
  tagline: 'x402 Meets libp2p. Money Flows Peer-to-Peer.',
  favicon: 'img/favicon.ico',

  url: 'https://asabya.github.io',
  baseUrl: '/betar/guide/',

  organizationName: 'asabya',
  projectName: 'betar',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
  },

  themes: ['@docusaurus/theme-mermaid'],

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          editUrl: 'https://github.com/asabya/betar/tree/master/docs-site/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'Betar',
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docsSidebar',
            position: 'left',
            label: 'Docs',
          },
          {
            href: 'https://github.com/asabya/betar',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Introduction',
                to: '/docs/intro',
              },
              {
                label: 'x402 Payments',
                to: '/docs/architecture/x402-payments',
              },
            ],
          },
          {
            title: 'Built On',
            items: [
              {
                label: 'libp2p',
                href: 'https://libp2p.io',
              },
              {
                label: 'IPFS',
                href: 'https://ipfs.tech',
              },
              {
                label: 'Protocol Labs',
                href: 'https://protocol.ai',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/asabya/betar',
              },
            ],
          },
        ],
        copyright: `Built for PL Genesis: Frontiers of Collaboration`,
      },
      prism: {
        theme: require('prism-react-renderer').themes.github,
        darkTheme: require('prism-react-renderer').themes.dracula,
        additionalLanguages: ['solidity', 'go', 'bash', 'yaml'],
      },
      mermaid: {
        theme: { light: 'neutral', dark: 'dark' },
      },
    }),
};

module.exports = config;
