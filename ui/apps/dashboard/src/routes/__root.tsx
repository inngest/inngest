/// <reference types="vite/client" />
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import * as React from 'react';

import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRouteWithContext,
  useLocation,
} from '@tanstack/react-router';

import CustomerIOAnalytics from '@/components/Analytics/CustomerIOAnalytics';
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

/**
 * Sign-up is always dark, regardless of the viewer's app theme: it is a
 * marketing surface with a fixed treatment, and most visitors arrive with no
 * stored preference anyway.
 *
 * Sign-in is deliberately NOT included. It is a returning-user surface, so
 * forcing it dark would flash every light-mode user on the way in and flash
 * them again when the app loads in their actual theme. The two live on separate
 * URLs, so they can differ.
 */
const FORCED_DARK_ROUTES = /^\/sign-up(\/|$)/;

function RootComponent() {
  const { pathname } = useLocation();

  // Sign-in and sign-up are marketing-adjacent and ship a fixed dark
  // treatment. `forcedTheme` overrides for these routes only and does not
  // write to the stored preference, so the rest of the app keeps whatever
  // theme the viewer chose.
  const forcedTheme = FORCED_DARK_ROUTES.test(pathname) ? 'dark' : undefined;

  return (
    <RootDocument>
      <ThemeProvider
        attribute="class"
        defaultTheme="system"
        forcedTheme={forcedTheme}
      >
        <InngestClerkProvider>
          <URQLProviderWrapper>
            <SentryUserIdentification />
            <ClientFeatureFlagProvider>
              <TooltipProvider delayDuration={0}>
                <Outlet />
              </TooltipProvider>

              <Toaster />
              <SegmentAnalytics />
              <CustomerIOAnalytics />
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
