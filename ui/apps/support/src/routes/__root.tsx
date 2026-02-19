/// <reference types="vite/client" />
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import * as React from "react";

import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRouteWithContext,
} from "@tanstack/react-router";

import fontsCss from "@inngest/components/AppRoot/fonts.css?url";
import globalsCss from "@inngest/components/AppRoot/globals.css?url";
import { TooltipProvider } from "@inngest/components/Tooltip";
import { ThemeProvider } from "next-themes";
import type { QueryClient } from "@tanstack/react-query";
import { StatusBanner } from "@inngest/components/Support/StatusBanner";
import type { ExtendedStatus } from "@inngest/components/Support/Status";
import { InngestClerkProvider } from "@/components/Clerk/Provider";
import { Navigation } from "@/components/Support/Navigation";
import { getStatus } from "@/data/status";

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
        title: "Inngest Support Portal",
        description: "The Inngest Support Portal",
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
        type: "image/svg+xml",
        href: import.meta.env.VITE_FAVICON ?? "/favicon.svg",
      },
      {
        rel: "icon",
        type: "image/png",
        sizes: "32x32",
        href: "/favicon-32x32.png",
      },
    ],
  }),

  loader: async () => {
    let status: ExtendedStatus | undefined = undefined; // Fetch system status
    try {
      status = await getStatus();
    } catch (error) {
      console.error("Failed to fetch status:", error);
    }
    return {
      status,
    };
  },
  component: RootComponent,
});

function RootComponent() {
  const { status } = Route.useLoaderData();

  return (
    <RootDocument>
      <ThemeProvider attribute="class" defaultTheme="system">
        <InngestClerkProvider>
          <TooltipProvider delayDuration={0}>
            <div className="flex min-h-screen flex-col md:flex-row bg-canvasBase">
              {/* Navigation */}
              <Navigation />

              <div className="flex flex-col grow">
                {/* Status Banner */}
                <StatusBanner status={status} />

                <div className="mx-auto w-full max-w-5xl py-6 px-4">
                  <Outlet />
                </div>
              </div>
            </div>
          </TooltipProvider>
        </InngestClerkProvider>
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
      <body className=" bg-canvasBase text-basis h-full overflow-auto overscroll-none">
        {children}
        <TanStackRouterDevtools position="bottom-right" />
        <Scripts />
      </body>
    </html>
  );
}
