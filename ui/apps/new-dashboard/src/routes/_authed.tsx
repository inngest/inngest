import { fetchClerkAuth } from "@/data/clerk";
import { createFileRoute, Outlet } from "@tanstack/react-router";
export const Route = createFileRoute("/_authed")({
  component: Authed,
  beforeLoad: async () => {
    const { userId, token } = await fetchClerkAuth();

    return {
      userId,
      token,
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
  return <Outlet />;
}
