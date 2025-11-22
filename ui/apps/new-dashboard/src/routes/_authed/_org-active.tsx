import Layout from "@/components/Layout/Layout";
import { navCollapsed } from "@/data/nav";
import { getEnvironment } from "@/queries/server-only/getEnvironment";
import { getProfileDisplay } from "@/queries/server-only/profile";
import { createFileRoute, notFound, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authed/_org-active")({
  component: OrgActive,

  head: () => ({
    //
    // TANSTACK TODO: initialize maze here
    scripts: [
      {
        src: "",
        type: "text/javascript",
      },
    ],
  }),
  loader: async ({ params }: { params: { envSlug?: string } }) => {
    const env = params.envSlug
      ? await getEnvironment({ data: { environmentSlug: params.envSlug } })
      : undefined;

    if (!env) {
      throw notFound({ data: { error: "Environment not found" } });
    }

    const profile = await getProfileDisplay();

    if (!profile) {
      throw notFound({ data: { error: "Profile not found" } });
    }

    return {
      env,
      profile,
      navCollapsed: await navCollapsed(),
    };
  },
});

function OrgActive() {
  const { env, navCollapsed, profile } = Route.useLoaderData();

  return (
    <Layout collapsed={navCollapsed} activeEnv={env} profile={profile}>
      <Outlet />
    </Layout>
  );
}
