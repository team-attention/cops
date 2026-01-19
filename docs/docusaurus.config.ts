import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const config: Config = {
  // Site metadata
  title: 'C-Ops Documentation',
  tagline: 'Claude Code Session Tracking System',
  favicon: 'img/favicon.ico',

  // Future flags, see https://docusaurus.io/docs/api/docusaurus-config#future
  future: {
    v4: true, // Improve compatibility with the upcoming Docusaurus v4
  },

  // Deployment URL configuration
  url: 'https://team-attention.github.io',
  baseUrl: '/cops/',

  // GitHub Pages deployment settings
  organizationName: 'team-attention',
  projectName: 'cops',
  trailingSlash: false,

  // Error handling for broken links
  onBrokenLinks: 'throw',

  // Markdown configuration
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  // i18n configuration - English as default
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
    localeConfigs: {
      en: {
        label: 'English',
        direction: 'ltr',
        htmlLang: 'en-US',
      },
    },
  },

  // Preset configuration
  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/team-attention/cops/edit/main/docs/',
          routeBasePath: '/',
          showLastUpdateTime: true,
          showLastUpdateAuthor: true,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  // Theme configuration
  themeConfig: {
    // SEO metadata
    metadata: [
      {name: 'description', content: 'C-Ops is a distributed system for tracking and visualizing Claude Code sessions across registered repositories.'},
      {name: 'keywords', content: 'Claude Code, session tracking, developer tools, documentation'},
      {name: 'og:title', content: 'C-Ops Documentation'},
      {name: 'og:description', content: 'Claude Code Session Tracking System'},
      {name: 'og:type', content: 'website'},
      {name: 'og:image', content: 'https://team-attention.github.io/cops/img/og-image.png'},
      {name: 'twitter:card', content: 'summary_large_image'},
    ],

    // Navbar configuration
    navbar: {
      title: 'C-Ops',
      logo: {
        alt: 'C-Ops Logo',
        src: 'img/logo.png',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          href: 'https://github.com/team-attention/cops',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },

    // Footer configuration
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Getting Started',
              to: '/getting-started/installation',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/team-attention/cops',
            },
          ],
        },
      ],
      copyright: `Copyright ${new Date().getFullYear()} Team Attention. Built with Docusaurus.`,
    },

    // Code highlighting theme
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'json', 'yaml'],
    },

    // Color mode configuration
    colorMode: {
      defaultMode: 'light',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
