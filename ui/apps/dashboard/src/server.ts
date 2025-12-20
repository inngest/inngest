import * as Sentry from '@sentry/tanstackstart-react';

const dsn = process.env.VITE_SENTRY_DSN;

if (dsn) {
  const release = process.env.VERCEL_GIT_COMMIT_SHA;
  const environment = process.env.VERCEL_ENV;

  Sentry.init({
    dsn,
    environment: environment ? `vercel-${environment}` : 'development',
    release,
    tracesSampleRate: 0.2,
  });
}

import handler from '@tanstack/react-start/server-entry';

export default handler;
