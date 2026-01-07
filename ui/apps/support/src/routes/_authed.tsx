import { Outlet, createFileRoute } from "@tanstack/react-router";
import { fetchClerkAuth } from "@/data/clerk";

export const Route = createFileRoute("/_authed")({
  component: Authed,
  beforeLoad: async () => {
    const { userId, token } = await fetchClerkAuth();

    return {
      userId,
      token,
    };
  },

  loader: () => {
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
