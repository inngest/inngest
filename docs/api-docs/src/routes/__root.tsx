import * as React from 'react';
import { HeadContent, Outlet, Scripts, createRootRoute } from '@tanstack/react-router';
import { RootProvider } from 'fumadocs-ui/provider/tanstack';

import '@/app.css';
import { SegmentPageTracking } from '@/components/SegmentPageTracking';
import { segmentLoaderSnippet } from '@/lib/analytics';

export const Route = createRootRoute({
  head: () => ({
    scripts: [{ children: segmentLoaderSnippet }],
  }),
  component: RootComponent,
});

function RootComponent() {
  return (
    <RootDocument>
      <Outlet />
    </RootDocument>
  );
}

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html suppressHydrationWarning lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <HeadContent />
      </head>
      <body className="flex min-h-screen flex-col">
        <RootProvider>{children}</RootProvider>
        <SegmentPageTracking />
        <Scripts />
      </body>
    </html>
  );
}
