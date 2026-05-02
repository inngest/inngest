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
  useNavigate,
  useRouterState,
} from '@tanstack/react-router';

import globalsCss from '@inngest/components/AppRoot/globals.css?url';
import fontsCss from '@inngest/components/AppRoot/fonts.css?url';
import StoreProvider from '@/components/StoreProvider';
import { useAuthStatusQuery } from '@/store/authApi';

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

function AuthGate({ children }: { children: React.ReactNode }) {
  const { data: authStatus, isLoading } = useAuthStatusQuery();
  const navigate = useNavigate();
  const routerState = useRouterState();
  const pathname = routerState.location.pathname;

  React.useEffect(() => {
    if (isLoading || !authStatus) return;
    if (
      authStatus.authRequired &&
      !authStatus.authenticated &&
      pathname !== '/login'
    ) {
      navigate({ to: '/login' });
    }
    if (authStatus.authenticated && pathname === '/login') {
      navigate({ to: '/' });
    }
  }, [authStatus, isLoading, pathname, navigate]);

  if (isLoading) {
    return (
      <div className="bg-canvasBase flex h-screen w-full items-center justify-center">
        <div className="text-muted text-sm">Loading...</div>
      </div>
    );
  }

  if (
    authStatus?.authRequired &&
    !authStatus.authenticated &&
    pathname !== '/login'
  ) {
    return null;
  }

  return <>{children}</>;
}

function RootComponent() {
  return (
    <RootDocument>
      <StoreProvider>
        <ThemeProvider attribute="class" defaultTheme="system">
          <AuthGate>
            <Outlet />
          </AuthGate>
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
