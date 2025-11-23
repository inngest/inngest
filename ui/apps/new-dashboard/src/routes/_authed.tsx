import { fetchClerkAuth } from "@/data/clerk";
import { createFileRoute, notFound, Outlet } from "@tanstack/react-router";

import Layout from "@/components/Layout/Layout";
import { navCollapsed } from "@/data/nav";
import { getEnvironment } from "@/queries/server-only/getEnvironment";
import { getProfileDisplay } from "@/queries/server-only/profile";

export const Route = createFileRoute("/_authed")({
  component: Authed,
  head: () => ({
    //
    // TANSTACK TODO: third party scripts to initial
    scripts: [
      {
        src: "",
        type: "text/javascript",
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
  errorComponent: ({ error }) => {
    if (error.message === "Not authenticated") {
      return "not authenticated";
    }

    //
    // TANSTACK TODO: render a nice shell layout
    throw error;
  },
});

function Authed() {
  const { env, navCollapsed, profile } = Route.useLoaderData();

  return (
    <Layout collapsed={navCollapsed} activeEnv={env} profile={profile}>
      <Outlet />
    </Layout>
  );
}
