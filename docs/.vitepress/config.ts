import { defineConfig, type HeadConfig } from 'vitepress'

const defaultTitle = 'LogLayer for Go'
const defaultDescription =
  'A transport-agnostic structured logging library for Go with a fluent API for messages, metadata, and errors.'
const baseUrl = 'https://go.loglayer.dev'

export default defineConfig({
  lang: 'en-US',
  title: 'LogLayer for Go',
  description: defaultDescription,
  srcDir: 'src',
  appearance: 'force-dark',
  sitemap: { hostname: baseUrl },
  async transformHead({ pageData }) {
    const head: HeadConfig[] = [
      ['link', { rel: 'icon', href: '/images/icons/favicon.ico' }],
      ['link', { rel: 'manifest', href: '/images/icons/site.webmanifest' }],
      ['meta', {
        name: 'keywords',
        content: 'loglayer, golang, go, logging, logger, structured, zerolog, zap',
      }],
      ['meta', { property: 'og:type', content: 'website' }],
      ['meta', { property: 'og:image', content: '/images/loglayer.png' }],
      ['meta', { property: 'og:site_name', content: 'LogLayer for Go' }],
      ['meta', { property: 'og:image:alt', content: 'LogLayer logo by Akshaya Madhavan' }],
      ['meta', { property: 'og:locale', content: 'en_US' }],
      ['meta', { name: 'twitter:card', content: 'summary' }],
      ['meta', { name: 'twitter:image:alt', content: 'LogLayer logo by Akshaya Madhavan' }],
    ]

    head.push([
      'meta',
      {
        property: 'og:title',
        content: String(pageData?.frontmatter?.title ?? defaultTitle).replace(/"/g, '&quot;'),
      },
    ])
    head.push([
      'meta',
      {
        property: 'og:description',
        content: String(pageData?.frontmatter?.description ?? defaultDescription).replace(
          /"/g,
          '&quot;'
        ),
      },
    ])
    head.push([
      'meta',
      {
        property: 'og:url',
        content: `${baseUrl}${pageData.relativePath ? '/' + pageData.relativePath.replace(/\.md$/, '') : ''}`,
      },
    ])

    return head
  },
  themeConfig: {
    logo: {
      src: '/images/loglayer.png',
      alt: 'LogLayer logo by Akshaya Madhavan',
    },
    editLink: {
      pattern: 'https://github.com/loglayer/loglayer-go/edit/main/docs/src/:path',
      text: 'Edit this page on GitHub',
    },
    search: { provider: 'local' },
    outline: { level: [2, 3] },
    nav: [
      { text: "What's New", link: '/whats-new' },
      { text: 'Get Started', link: '/getting-started' },
      { text: 'Logging API', link: '/logging-api/basic-logging' },
      { text: 'Transports', link: '/transports/' },
      { text: 'Integrations', link: '/integrations/loghttp' },
    ],
    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: "What's New", link: '/whats-new' },
          { text: 'Why Use LogLayer?', link: '/introduction' },
          { text: 'Getting Started', link: '/getting-started' },
          { text: 'Configuration', link: '/configuration' },
          { text: 'Cheat Sheet', link: '/cheatsheet' },
          { text: 'Benchmarks', link: '/benchmarks' },
        ],
      },
      {
        text: 'Logging API',
        items: [
          { text: 'Basic Logging', link: '/logging-api/basic-logging' },
          { text: 'Adjusting Log Levels', link: '/logging-api/adjusting-log-levels' },
          { text: 'Fields', link: '/logging-api/fields' },
          { text: 'Go Context', link: '/logging-api/go-context' },
          { text: 'Metadata', link: '/logging-api/metadata' },
          { text: 'Error Handling', link: '/logging-api/error-handling' },
          { text: 'Child Loggers', link: '/logging-api/child-loggers' },
          { text: 'Transport Management', link: '/logging-api/transport-management' },
          { text: 'Raw Logging', link: '/logging-api/raw' },
          { text: 'Mocking', link: '/logging-api/mocking' },
        ],
      },
      {
        text: 'Integrations',
        items: [
          { text: 'HTTP Middleware (loghttp)', link: '/integrations/loghttp' },
        ],
      },
      {
        text: 'Transports',
        items: [
          { text: 'Overview', link: '/transports/' },
          { text: 'Multiple Transports', link: '/transports/multiple-transports' },
          { text: 'Creating Transports', link: '/transports/creating-transports' },
          {
            text: 'Renderers',
            items: [
              { text: 'Blank', link: '/transports/blank' },
              { text: 'Console', link: '/transports/console' },
              { text: 'Pretty', link: '/transports/pretty' },
              { text: 'Structured', link: '/transports/structured' },
              { text: 'Testing', link: '/transports/testing' },
            ],
          },
          {
            text: 'Logger Wrappers',
            items: [
              { text: 'Zerolog', link: '/transports/zerolog' },
              { text: 'Zap', link: '/transports/zap' },
              { text: 'log/slog', link: '/transports/slog' },
              { text: 'phuslu/log', link: '/transports/phuslu' },
              { text: 'logrus', link: '/transports/logrus' },
              { text: 'charmbracelet/log', link: '/transports/charmlog' },
            ],
          },
        ],
      },
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/loglayer/loglayer-go' },
    ],
  },
})
