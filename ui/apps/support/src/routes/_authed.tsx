import Layout from "@/components/Layout/Layout";
import { fetchClerkAuth } from "@/data/clerk";
import { navCollapsed } from "@/data/nav";
import { getProfileDisplay } from "@/data/profile";
import { createFileRoute, notFound, Outlet } from "@tanstack/react-router";
export const Route = createFileRoute("/_authed")({
  component: Authed,
  beforeLoad: async () => {
    const { userId, token } = await fetchClerkAuth();

    return {
      userId,
      token,
    };
  },

  loader: async () => {
    const profile = await getProfileDisplay();

    if (!profile) {
      throw notFound({ data: { error: "Profile not found" } });
    }

    return {
      profile,
      navCollapsed: await navCollapsed(),
    };
  },

  errorComponent: ({ error }) => {
    if (error.message === "Not authenticated") {
      return "not authenticated";
    }

    throw error;
  },
});

function Authed() {
  const { navCollapsed, profile } = Route.useLoaderData();

  return (
    <Layout collapsed={navCollapsed} profile={profile}>
      <Outlet />
    </Layout>
  );
}
