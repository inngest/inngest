import { ArchivedEnvBanner } from "@/components/ArchivedEnvBanner";
import { EnvironmentProvider } from "@/components/Environments/environment-context";
import { Alert } from "@inngest/components/Alert";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { Route as OrgActiveRoute } from "@/routes/_authed/_org-active";

export const Route = createFileRoute("/_authed/_org-active/env/$envSlug")({
  component: EnvLayout,
});

const NotFound = () => (
  <div className="mt-16 flex place-content-center">
    <Alert severity="warning">Environment not found.</Alert>
  </div>
);

function EnvLayout() {
  const { env } = OrgActiveRoute.useLoaderData();

  return env ? (
    <>
      <ArchivedEnvBanner env={env} />
      <EnvironmentProvider env={env}>
        <Outlet />
      </EnvironmentProvider>
    </>
  ) : (
    <NotFound />
  );
}
