import * as Sentry from '@sentry/tanstackstart-react';

Sentry.init({
  debug: true,
  dsn: process.env.VITE_SENTRY_DSN,
  tracesSampleRate: 0.2,
});
