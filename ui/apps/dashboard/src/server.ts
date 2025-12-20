import * as Sentry from '@sentry/tanstackstart-react';

const dsn = process.env.VITE_SENTRY_DSN;

console.log('server side sentry dsn', dsn);

if (dsn) {
  const release = process.env.VITE_VERCEL_GIT_COMMIT_SHA;
  const environment = process.env.VITE_VERCEL_ENV;

  console.log('server side sentry release', release);
  console.log('server side sentry environment', environment);

  Sentry.init({
    dsn,
    environment,
    release,
    tracesSampleRate: 0.2,
  });
}

import handler from '@tanstack/react-start/server-entry';

export default handler;
