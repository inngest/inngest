import { createFileRoute, Outlet } from '@tanstack/react-router';

import { DuckDBGuard } from '@/components/DuckDBGuard';

// Guards every /sessions route (keys, a key's sessions, a session's runs):
// they all query resolvers that need --duckdb.
export const Route = createFileRoute('/_dashboard/sessions')({
  component: SessionsLayout,
});

function SessionsLayout() {
  return (
    <DuckDBGuard>
      <Outlet />
    </DuckDBGuard>
  );
}
