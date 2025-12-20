import * as Sentry from '@sentry/tanstackstart-react';

const dsn = process.env.VITE_SENTRY_DSN;

if (dsn) {
  const isDev = process.env.NODE_ENV === 'development';

  const release =
    process.env.VITE_SENTRY_RELEASE ||
    process.env.VERCEL_GIT_COMMIT_SHA ||
    process.env.VITE_VERCEL_GIT_COMMIT_SHA ||
    undefined;

  const environment =
    process.env.VITE_ENVIRONMENT ||
    process.env.VERCEL_ENV ||
    process.env.VITE_VERCEL_ENV ||
    process.env.NODE_ENV;

  Sentry.init({
    debug: isDev,
    dsn,
    environment,
    release,
    tracesSampleRate: 0.2,
  });
}

import handler from '@tanstack/react-start/server-entry';

export default handler;
