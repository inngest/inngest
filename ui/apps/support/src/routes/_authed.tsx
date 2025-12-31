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

  loader: async () => {
    return {};
  },

  errorComponent: (props) => {
    if (props.error.message === "Not authenticated") {
      return "not authenticated";
    }
    console.error(props.error);

    return <div>{props.error.message}</div>;
  },
});

function Authed() {
  return <Outlet />;
}
