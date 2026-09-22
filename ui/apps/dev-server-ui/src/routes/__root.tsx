/// <reference types="vite/client" />
import * as React from 'react';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { ThemeProvider } from 'next-themes';
import { Toaster } from 'sonner';

import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRoute,
} from '@tanstack/react-router';

import globalsCss from '@inngest/components/AppRoot/globals.css?url';
import fontsCss from '@inngest/components/AppRoot/fonts.css?url';
import StoreProvider from '@/components/StoreProvider';
import { HostedConnection } from '@/components/HostedConnection';
import { isHosted } from '@/utils/devServer';

export const Route = createRootRoute({
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
        title: isHosted
          ? 'Inngest Dev Server | Local workflow development'
          : 'Inngest Server',
      },
      ...(isHosted
        ? [
            {
              name: 'description',
              content:
                'Build and debug durable workflows locally. Connect to your Inngest Dev Server to send events, inspect function runs, and explore execution traces.',
            },
            { name: 'robots', content: 'noindex, follow' },
            { property: 'og:title', content: 'Inngest Dev Server' },
            {
              property: 'og:description',
              content:
                'Build and debug durable workflows on your machine from your browser.',
            },
            { property: 'og:type', content: 'website' },
            {
              property: 'og:image',
              content:
                'https://www.inngest.com/assets/homepage/open-graph-2026.png',
            },
            { name: 'twitter:card', content: 'summary_large_image' },
          ]
        : []),
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
        href: `${import.meta.env.BASE_URL}favicon-june-2025.svg`,
        media: '(prefers-color-scheme: light)',
      },
      {
        rel: 'icon',
        href: `${import.meta.env.BASE_URL}favicon-june-2025.svg`,
        media: '(prefers-color-scheme: dark)',
      },
    ],
  }),
  component: RootComponent,
});

function RootComponent() {
  const Connection = isHosted ? HostedConnection : React.Fragment;
  return (
    <RootDocument>
      <ThemeProvider attribute="class" defaultTheme="system">
        <Connection>
          <StoreProvider>
            <Outlet />
            <Toaster
              toastOptions={{
                className: 'drop-shadow-lg',
                style: {
                  background: `rgb(var(--color-background-canvas-base))`,
                  borderRadius: 0,
                  borderWidth: '0px 0px 2px',
                  color: `rgb(var(--color-foreground-base))`,
                },
              }}
            />
          </StoreProvider>
        </Connection>
      </ThemeProvider>
    </RootDocument>
  );
}

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="h-full" suppressHydrationWarning>
      <head>
        <HeadContent />
      </head>
      <body className="bg-canvasBase text-basis h-full overflow-auto overscroll-none">
        <div id="app" />
        <div id="modals" />
        {children}
        {import.meta.env.DEV && (
          <TanStackRouterDevtools position="bottom-right" />
        )}
        <Scripts />
      </body>
    </html>
  );
}
