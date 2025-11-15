import { createFileRoute, Outlet } from "@tanstack/react-router";
export const Route = createFileRoute("/_authed")({
  component: Authed,
  beforeLoad: ({ context }) => {
    if (!context.userId) {
      throw new Error("Not authenticated");
    }
  },
  loader: async () => {
    // return {
    //   navCollapsed: await navCollapsed(),
    // };
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
