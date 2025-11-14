import * as React from "react";
import { createRouter } from "@tanstack/react-router";
import { setupRouterSsrQueryIntegration } from "@tanstack/react-router-ssr-query";

import { routeTree } from "./routeTree.gen";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const QueryWrapper = ({
  queryClient,
  children,
}: {
  queryClient: QueryClient;
  children: React.ReactNode;
}) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
);

export const getRouter = () => {
  const queryClient = new QueryClient();

  const router = createRouter({
    routeTree,
    context: { queryClient },
    defaultPreload: "intent",
    defaultErrorComponent: (err) => <p>{err.error.stack}</p>,
    defaultNotFoundComponent: () => <p>not found</p>,
    Wrap: (props: { children: React.ReactNode }) => (
      <QueryWrapper queryClient={queryClient}>{props.children}</QueryWrapper>
    ),
  });

  setupRouterSsrQueryIntegration({
    router,
    queryClient,
  });

  return router;
};
