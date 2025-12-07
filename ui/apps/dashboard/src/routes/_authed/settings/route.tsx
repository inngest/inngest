import { createFileRoute, Outlet } from "@tanstack/react-router";

import { SettingsHeader } from "@/components/Settings/SettingsHeader";

export const Route = createFileRoute("/_authed/settings")({
  component: SettingsLayout,
});

function SettingsLayout() {
  return (
    <div className="h-full flex-col">
      <SettingsHeader />
      <div className="no-scrollbar h-full overflow-y-scroll px-6">
        <Outlet />
      </div>
    </div>
  );
}
