import { ArchivedEnvBanner } from "@/components/Environments/ArchivedEnvBanner";
import { EnvironmentProvider } from "@/components/Environments/environment-context";
import { SharedContextProvider } from "@/components/SharedContext/SharedContextProvider";
import { Alert } from "@inngest/components/Alert/NewAlert";
import { createFileRoute, notFound, Outlet } from "@tanstack/react-router";

import { getEnvironment } from "@/queries/server/getEnvironment";

const NotFound = () => (
  <div className="mt-16 flex place-content-center">
    <Alert severity="warning">Environment not found.</Alert>
  </div>
);

export const Route = createFileRoute("/_authed/env/$envSlug")({
  component: EnvLayout,
  notFoundComponent: NotFound,
  loader: async ({ params }) => {
    const env = await getEnvironment({
      data: { environmentSlug: params.envSlug },
    });

    if (params.envSlug && !env) {
      throw notFound({ data: { error: "Environment not found" } });
    }

    return {
      env,
    };
  },
});

function EnvLayout() {
  const { env } = Route.useLoaderData();

  return (
    <>
      <ArchivedEnvBanner env={env} />
      <EnvironmentProvider env={env}>
        <SharedContextProvider>
          <Outlet />
        </SharedContextProvider>
      </EnvironmentProvider>
    </>
  );
}
