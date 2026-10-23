import { createFileRoute, Outlet } from '@tanstack/react-router';

import { SettingsHeader } from '@/components/Settings/SettingsHeader';

export const Route = createFileRoute('/_authed/settings')({
  component: SettingsLayout,
});

function SettingsLayout() {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <SettingsHeader />
      <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto px-6">
        <Outlet />
      </div>
    </div>
  );
}
