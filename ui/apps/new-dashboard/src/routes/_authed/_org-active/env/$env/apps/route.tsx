import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authed/_org-active/env/$env/apps")({
  component: AppLayout,
});

function AppLayout() {
  const { env: envSlug } = Route.useParams();

  return (
    <div className="flex flex-col">
      app layout coming soon
      <Outlet />
    </div>
  );
}
