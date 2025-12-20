import * as React from 'react';
import { createRouter } from '@tanstack/react-router';
import { setupRouterSsrQueryIntegration } from '@tanstack/react-router-ssr-query';

import { routeTree } from './routeTree.gen';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import NotFound from './components/Error/NotFound';
import { SentryWrappedCatchBoundary } from './components/Error/DefaultCatchBoundary';

import * as Sentry from '@sentry/tanstackstart-react';

export const getRouter = () => {
  const queryClient = new QueryClient();

  const router = createRouter({
    routeTree,
    context: { queryClient },
    defaultPreload: 'intent',
    defaultErrorComponent: SentryWrappedCatchBoundary,
    defaultNotFoundComponent: () => <NotFound />,
    Wrap: (props: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        {props.children}
      </QueryClientProvider>
    ),
  });

  setupRouterSsrQueryIntegration({
    router,
    queryClient,
  });

  if (!router.isServer && !Sentry.getClient()) {
    const dsn = import.meta.env.VITE_SENTRY_DSN;

    if (dsn) {
      const isDev = import.meta.env.DEV;

      const release =
        import.meta.env.VITE_SENTRY_RELEASE ||
        import.meta.env.VITE_VERCEL_GIT_COMMIT_SHA ||
        (isDev ? `dev-${Date.now()}` : undefined);

      const environment =
        import.meta.env.VITE_ENVIRONMENT ||
        import.meta.env.VITE_VERCEL_ENV ||
        import.meta.env.MODE;

      Sentry.init({
        debug: isDev,
        dsn,
        environment,
        release,
        tracesSampleRate: 0.2,
        tracePropagationTargets: [
          /^\//, // All URLs on current origin.
          /^https:\/\/api\.inngest\.com\//, // The production API origin.
          /^https:\/\/api\.inngest\.net\//, // The staging API origin.
          'localhost', // The local API origin.
        ],
        replaysSessionSampleRate: 0.2,
        replaysOnErrorSampleRate: 1.0,
        integrations: [
          Sentry.replayIntegration({
            maskAllText: false,
            blockAllMedia: false,
          }),
        ],
      });
    }
  }

  return router;
};
