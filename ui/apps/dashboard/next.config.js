// @ts-check
const { withSentryConfig } = require('@sentry/nextjs');

/** @type {import('next').NextConfig} */
const nextConfig = {
  productionBrowserSourceMaps: true,
  experimental: {
    turbo: {
      rules: {
        '*.svg': {
          loaders: ['@svgr/webpack'],
          as: '*.js',
        },
      },
    },
  },
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'img.clerk.com',
      },
    ],
  },
  transpilePackages: ['@inngest/components'],
  async redirects() {
    return [
      {
        source: '/',
        destination: '/env/production/apps',
        permanent: false,
      },
      {
        source: '/env/:slug/manage',
        destination: '/env/:slug/manage/keys',
        permanent: false,
      },
      {
        source: '/integrations/vercel',
        destination: '/integrations/vercel/callback',
        permanent: false,
      },
      {
        source: '/login',
        destination: '/sign-in',
        permanent: false,
      },
      {
        source: '/reset-password/reset',
        destination: '/sign-in',
        permanent: false,
      },
      // Legacy Pages
      {
        source: '/env/:slug/deploys',
        destination: '/env/:slug/apps',
        permanent: false,
      },
      {
        source: '/settings/team',
        destination: '/settings/organization',
        permanent: false,
      },
      // Legacy signing key locations
      {
        source: '/secrets',
        destination: '/env/production/manage/signing-key',
        permanent: false,
      },
      {
        source: '/test/secrets',
        destination: '/env/branch/manage/signing-key',
        permanent: false,
      },
    ];
  },
  // Optional build-time configuration for Sentry.
  // See https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/#extend-nextjs-configuration
  sentry: {
    hideSourceMaps: false,
    // Tunnel sentry events to help circumvent ad-blockers.
    tunnelRoute: '/api/sentry',
  },
};

/**
 * Additional config options for the Sentry Webpack plugin. Keep in mind that
 * the following options are set automatically, and overriding them is not
 * recommended:
 *   release, url, org, project, authToken, configFile, stripPrefix,
 *   urlPrefix, include, ignore
 *
 * For all available options:
 * @see {@link https://github.com/getsentry/sentry-webpack-plugin#options}
 */
const sentryWebpackPluginOptions = {
  silent: true,
};

// Make sure adding Sentry options is the last code to run before exporting, to
// ensure that your source maps include changes from all other Webpack plugins
module.exports = withSentryConfig(nextConfig, sentryWebpackPluginOptions);
