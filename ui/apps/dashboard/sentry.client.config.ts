// This file configures the initialization of Sentry on the browser.
// The config you add here will be used whenever a page is visited.
// https://docs.sentry.io/platforms/javascript/guides/nextjs/

import * as Sentry from '@sentry/nextjs';

Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  tracesSampleRate: 0.2,
  tracePropagationTargets: [
    /^\//, // All URLs on current origin.
    /^https:\/\/api\.inngest\.com\//, // The production API origin.
    /^https:\/\/api\.inngest\.net\//, // The staging API origin.
    'localhost', // The local API origin.
  ],
  replaysSessionSampleRate: 0.0,
  replaysOnErrorSampleRate: 1.0,
});

Sentry.getCurrentHub?.()
  ?.getClient?.()
  ?.addIntegration?.(
    new Sentry.Replay({
      maskAllText: false,
      blockAllMedia: false,
    })
  );
