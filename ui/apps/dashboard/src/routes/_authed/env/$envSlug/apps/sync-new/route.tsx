import { Header } from '@inngest/components/Header/Header';
import { createFileRoute, Outlet } from '@tanstack/react-router';

import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/apps/sync-new')({
  component: SyncNewLayout,
});

function SyncNewLayout() {
  const { envSlug } = Route.useParams();

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Apps', href: pathCreator.apps({ envSlug }) },
          { text: 'Sync new' },
        ]}
      />
      <Outlet />
    </>
  );
}
