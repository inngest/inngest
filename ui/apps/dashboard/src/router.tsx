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

  if (!router.isServer) {
    Sentry.init({
      dsn: import.meta.env.VITE_SENTRY_DSN,
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

  return router;
};
