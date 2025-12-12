import { Header } from '@inngest/components/Header/NewHeader';
import { createFileRoute, Outlet } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/env/$envSlug/unattached-syncs')({
  component: UnattachedSyncsLayout,
});

function UnattachedSyncsLayout() {
  return (
    <>
      <Header breadcrumb={[{ text: 'Unattached Syncs', href: '/env' }]} />

      <div className="no-scrollbar h-full overflow-y-scroll">
        <Outlet />
      </div>
    </>
  );
}
