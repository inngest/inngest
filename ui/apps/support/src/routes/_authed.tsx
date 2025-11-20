import Layout from "@/components/Layout/Layout";
import { fetchClerkAuth } from "@/data/clerk";
import { navCollapsed } from "@/data/nav";
import { getProfileDisplay } from "@/data/profile";
import { Header } from "@inngest/components/Header/NewHeader";
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

  errorComponent: (props) => {
    if (props.error.message === "Not authenticated") {
      return "not authenticated";
    }
    console.error(props.error);

    return (
      //
      // TODO: handle "inngest is down" error specifically
      <Layout collapsed={false}>
        <Header breadcrumb={[{ text: "Support" }]} />
        <div className="m-8 flex flex-col gap-2">file a ticket</div>
      </Layout>
    );
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
