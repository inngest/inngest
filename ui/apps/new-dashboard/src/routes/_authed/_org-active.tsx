import Layout from "@/components/Layout/Layout";
import { navCollapsed } from "@/data/nav";
import { getEnvironment } from "@/queries/server-only/getEnvironment";
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
  loader: async ({ params }: { params: { env?: string } }) => {
    const env = params.env
      ? await getEnvironment({ data: { environmentSlug: params.env } })
      : undefined;

    if (!env) {
      throw notFound();
    }
    return {
      env,
      navCollapsed: await navCollapsed(),
    };
  },
});

function OrgActive() {
  const { navCollapsed, env } = Route.useLoaderData();

  return (
    <Layout collapsed={navCollapsed} activeEnv={env} profile={undefined}>
      <Outlet />
    </Layout>
  );
}
