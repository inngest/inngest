import * as Sentry from '@sentry/tanstackstart-react';

console.log('[Sentry Server] Instrumentation file loaded');

const dsn = process.env.VITE_SENTRY_DSN;

if (dsn) {
  console.log(
    '[Sentry Server] Initializing with DSN:',
    dsn.substring(0, 30) + '...',
  );
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
    debug: true,
    dsn,
    environment,
    release,
    tracesSampleRate: 0.2,
  });

  console.log('[Sentry Server] Initialized with:', { environment, release });
} else {
  console.log('[Sentry Server] Skipping initialization - no DSN found');
}
