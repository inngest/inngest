import type { ReactNode } from 'react';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { DIContext } from '@inngest/components/contexts/di';
import { HeadContent, Outlet, createRootRoute, useLocation } from '@tanstack/react-router';

import appCss from './global.css?url';

export const Route = createRootRoute({
  head: () => ({
    links: [{ rel: 'stylesheet', href: appCss }],
    meta: [
      {
        charSet: 'utf-8',
      },
      {
        name: 'viewport',
        content: 'width=device-width, initial-scale=1',
      },
      {
        title: 'TanStack Start Starter',
      },
    ],
  }),
  component: Component,
});

function Component() {
  return (
    <RootDocument>
      <Outlet />
    </RootDocument>
  );
}

function RootDocument({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <html>
      <head>
        <HeadContent />
      </head>
      <body>
        <DIContext.Provider value={{ usePathname }}>
          <TooltipProvider delayDuration={0}>{children}</TooltipProvider>
        </DIContext.Provider>
      </body>
    </html>
  );
}

function usePathname() {
  const location = useLocation();
  return location.pathname;
}
