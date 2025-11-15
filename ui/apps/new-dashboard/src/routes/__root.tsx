/// <reference types="vite/client" />
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import * as React from "react";

import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRouteWithContext,
} from "@tanstack/react-router";

import { InngestClerkProvider } from "@/components/Clerk/Provider";
import fontsCss from "@inngest/components/AppRoot/fonts.css?url";
import globalsCss from "@inngest/components/AppRoot/globals.css?url";
import { ThemeProvider } from "next-themes";
import { ClientFeatureFlagProvider } from "@/components/FeatureFlags/ClientFeatureFlagProvider";
import { TooltipProvider } from "@inngest/components/Tooltip";
import { fetchClerkAuth } from "@/data/clerk";
import { navCollapsed } from "@/data/nav";
import { QueryClient } from "@tanstack/react-query";

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
}>()({
  head: () => ({
    meta: [
      {
        charSet: "utf-8",
      },
      {
        name: "viewport",
        content: "width=device-width, initial-scale=1",
      },
      {
        title: "Inngest Dashboard",
        description: "The Inngest Cloud dashboard",
      },
    ],

    links: [
      {
        rel: "stylesheet",
        href: globalsCss,
      },
      {
        rel: "stylesheet",
        href: fontsCss,
      },
      {
        rel: "icon",
        url: "/favicon-june-2025-light.svg",
        media: "(prefers-color-scheme: light)",
      },
      {
        rel: "icon",
        url: "/favicon-june-2025-dark.svg",
        media: "(prefers-color-scheme: dark)",
      },
    ],
  }),
  beforeLoad: async () => {
    const { userId, token } = await fetchClerkAuth();

    return {
      userId,
      token,
    };
  },
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
          {/* TANSTACK TODO: add sentry user identification provider here */}
          <ClientFeatureFlagProvider>
            <TooltipProvider delayDuration={0}>
              <Outlet />
            </TooltipProvider>
            {/* TANSTACK TODO: add page view tracker here */}
          </ClientFeatureFlagProvider>
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
