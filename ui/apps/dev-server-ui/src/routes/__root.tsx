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
        title: 'Inngest Server',
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
        href: '/favicon-june-2025.svg',
        media: '(prefers-color-scheme: light)',
      },
      {
        rel: 'icon',
        href: '/favicon-june-2025.svg',
        media: '(prefers-color-scheme: dark)',
      },
    ],
  }),
  component: RootComponent,
});

// @monaco-editor/react's useMonaco() hook (used by SQLEditor's
// useMonacoWithTheme) doesn't attach a .catch() to the cancelable promise
// its own mount effect creates, unlike its <Editor> component (which
// does) -- unmounting a useMonaco()-using component before the shared
// Monaco script finishes loading always surfaces this exact shape as an
// unhandled rejection. It's expected and harmless (loader.init()'s
// underlying promise is a shared singleton -- one consumer canceling its
// own wrapper doesn't affect any other's), and can't be fixed without
// patching that library, so it's filtered globally here instead of
// alarming users/monitoring for a library-internal non-issue.
function isMonacoLoaderCancelation(reason: unknown): boolean {
  return (
    typeof reason === 'object' &&
    reason !== null &&
    (reason as Record<string, unknown>).type === 'cancelation' &&
    (reason as Record<string, unknown>).msg === 'operation is manually canceled'
  );
}

function RootComponent() {
  React.useEffect(() => {
    const handleRejection = (e: PromiseRejectionEvent) => {
      if (isMonacoLoaderCancelation(e.reason)) {
        e.preventDefault();
      }
    };
    window.addEventListener('unhandledrejection', handleRejection);
    return () =>
      window.removeEventListener('unhandledrejection', handleRejection);
  }, []);

  return (
    <RootDocument>
      <StoreProvider>
        <ThemeProvider attribute="class" defaultTheme="system">
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
        </ThemeProvider>
      </StoreProvider>
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
        <TanStackRouterDevtools position="bottom-right" />
        <Scripts />
      </body>
    </html>
  );
}
