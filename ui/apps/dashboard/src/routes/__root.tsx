/// <reference types="vite/client" />
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import * as React from 'react';

import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRouteWithContext,
} from '@tanstack/react-router';

import SegmentAnalytics from '@/components/Analytics/SegmentAnalytics';
import SentryUserIdentification from '@/components/Analytics/SentryUserIdentification';
import { InngestClerkProvider } from '@/components/Clerk/Provider';
import { ClientFeatureFlagProvider } from '@/components/FeatureFlags/ClientFeatureFlagProvider';
import Toaster from '@/components/Toast/Toaster';
import URQLProviderWrapper from '@/components/URQL/URQLProvider';
import { navCollapsed } from '@/lib/nav';
import fontsCss from '@inngest/components/AppRoot/fonts.css?url';
import globalsCss from '@inngest/components/AppRoot/globals.css?url';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { QueryClient } from '@tanstack/react-query';
import { ThemeProvider } from 'next-themes';

//
// don't load locally, causes issues with adblockers
const PageViewTracker = React.lazy(() =>
  import.meta.env.PROD
    ? import('@/components/Analytics/PageViewTracker')
    : Promise.resolve({ default: () => null }),
);

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
}>()({
  head: () => ({
    meta: [
      {
        charSet: 'utf-8',
      },
      {
        name: 'viewport',
        content: 'width=device-width, initial-scale=1',
      },
      {
        title: 'Inngest Dashboard',
        description: 'The Inngest Cloud dashboard',
      },
    ],

    links: [
      {
        rel: 'stylesheet',
        href: globalsCss,
      },
      {
        rel: 'stylesheet',
        href: fontsCss,
      },
      {
        rel: 'icon',
        type: 'image/svg+xml',
        href: import.meta.env.VITE_FAVICON ?? '/favicon.svg',
      },
      {
        rel: 'icon',
        type: 'image/png',
        sizes: '32x32',
        href: '/favicon-32x32.png',
      },
      {
        rel: 'apple-touch-icon',
        sizes: '180x180',
        href: '/apple-touch-icon.png',
      },
    ],
  }),

  loader: async () => {
    return {
      navCollapsed: await navCollapsed(),
    };
  },
  component: RootComponent,
});

function RootComponent() {
  return (
    <RootDocument>
      <ThemeProvider attribute="class" defaultTheme="system">
        <InngestClerkProvider>
          <URQLProviderWrapper>
            <SentryUserIdentification />
            <ClientFeatureFlagProvider>
              <TooltipProvider delayDuration={0}>
                <Outlet />
              </TooltipProvider>

              <Toaster />
              <SegmentAnalytics />
              <React.Suspense>
                <PageViewTracker />
              </React.Suspense>
            </ClientFeatureFlagProvider>
          </URQLProviderWrapper>
        </InngestClerkProvider>
      </ThemeProvider>
    </RootDocument>
  );
}

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="h-full">
      <head>
        <HeadContent />
      </head>
      <body className=" bg-canvasBase text-basis h-full overflow-auto overscroll-none">
        <div id="app" />
        <div id="modals" />
        {children}
        <TanStackRouterDevtools position="bottom-right" />
        <Scripts />
      </body>
    </html>
  );
}
