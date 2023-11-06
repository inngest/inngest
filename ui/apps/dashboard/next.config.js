// @ts-check
const { withSentryConfig } = require('@sentry/nextjs');

/** @type {import('next').NextConfig} */
const nextConfig = {
  experimental: {
    typedRoutes: true,
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
        destination: '/env/production/functions',
        permanent: true,
      },
      {
        source: '/env/:slug/manage',
        destination: '/env/:slug/manage/keys',
        permanent: true,
      },
      {
        source: '/env/:slug/settings',
        destination: '/env/:slug/settings/team',
        permanent: true,
      },
      {
        source: '/integrations/vercel',
        destination: '/integrations/vercel/callback',
        permanent: false,
      },
      {
        source: '/settings/integrations',
        destination: '/settings/integrations/vercel',
        permanent: false,
      },
      {
        source: '/login',
        destination: '/sign-in',
        permanent: true,
      },
      {
        source: '/reset-password/reset',
        destination: '/password-reset/complete',
        permanent: true,
      },
      // Legacy signing key locations
      {
        source: '/secrets',
        destination: '/env/production/manage/signing-key',
        permanent: true,
      },
      {
        source: '/test/secrets',
        destination: '/env/branch/manage/signing-key',
        permanent: true,
      },
    ];
  },
  // Optional build-time configuration for Sentry.
  // See https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/#extend-nextjs-configuration
  sentry: {
    // Use `hidden-source-map` rather than `source-map` as the Webpack `devtool`
    // for client-side builds. (This will be the default starting in
    // `@sentry/nextjs` version 8.0.0.) See
    // https://webpack.js.org/configuration/devtool/ and
    // https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/#use-hidden-source-map
    // for more information.
    hideSourceMaps: true,
    // Tunnel sentry events to help circumvent ad-blockers.
    tunnelRoute: '/api/sentry',
  },
  webpack(config, { isServer }) {
    // Configures webpack to handle SVG files with SVGR. SVGR optimizes and transforms SVG files
    // into React components. See https://react-svgr.com/docs/next/

    // Grab the existing rule that handles SVG imports
    // @ts-ignore - this is a private property that is not typed
    const fileLoaderRule = config.module.rules.find((rule) => rule.test?.test?.('.svg'));

    config.module.rules.push(
      // Reapply the existing rule, but only for svg imports ending in ?url
      {
        ...fileLoaderRule,
        test: /\.svg$/i,
        resourceQuery: /url/, // *.svg?url
      },
      // Convert all other *.svg imports to React components
      {
        test: /\.svg$/i,
        issuer: fileLoaderRule.issuer,
        resourceQuery: { not: [...fileLoaderRule.resourceQuery.not, /url/] }, // exclude if *.svg?url
        use: ['@svgr/webpack'],
      }
    );

    // Modify the file loader rule to ignore *.svg, since we have it handled now.
    fileLoaderRule.exclude = /\.svg$/i;

    // If client-side, don't polyfill `fs`.
    // This is needed for `quickjs-emscripten` to work on the client. See https://github.com/justjake/quickjs-emscripten/issues/33#issuecomment-739098440
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
      };
    }

    return config;
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
