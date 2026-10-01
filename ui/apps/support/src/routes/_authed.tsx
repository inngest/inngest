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
    const message =
      props.error instanceof Error ? props.error.message : String(props.error);
    if (message === "Not authenticated") {
      return "not authenticated";
    }
    console.error(props.error);

    return <div>{message}</div>;
  },
});

function Authed() {
  return <Outlet />;
}
