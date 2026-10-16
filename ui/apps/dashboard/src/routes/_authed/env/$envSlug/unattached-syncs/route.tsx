import { Header } from '@inngest/components/Header/Header';
import { createFileRoute, Outlet } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/env/$envSlug/unattached-syncs')({
  component: UnattachedSyncsLayout,
});

function UnattachedSyncsLayout() {
  return (
    <>
      <Header breadcrumb={[{ text: 'Unattached Syncs', href: '/env' }]} />

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <Outlet />
      </div>
    </>
  );
}
