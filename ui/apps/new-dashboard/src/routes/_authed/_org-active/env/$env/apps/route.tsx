import { createFileRoute, Outlet } from "@tanstack/react-router";
import { HeaderType } from "@inngest/components/Header/NewHeader";

export const Route = createFileRoute("/_authed/_org-active/env/$env/apps")({
  component: AppLayout,
  beforeLoad: () => {
    return {
      layoutHeader: {
        breadcrumb: [{ text: "Apps" }],
        backNav: true,
      } satisfies HeaderType,
    };
  },
});

function AppLayout() {
  return <Outlet />;
}
