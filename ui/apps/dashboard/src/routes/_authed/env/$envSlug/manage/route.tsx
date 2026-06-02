import { createFileRoute, Outlet } from '@tanstack/react-router';

import { useEnvironment } from '@/components/Environments/environment-context';
import ChildEmptyState from '@/components/Manage/ChildEmptyState';
import { ManageHeader } from '@/components/Manage/Header';

export const Route = createFileRoute('/_authed/env/$envSlug/manage')({
  component: ManageLayoutComponent,
});

function ManageLayoutComponent() {
  const env = useEnvironment();

  if (env?.hasParent) {
    return <ChildEmptyState />;
  }

  return (
    <>
      <ManageHeader />
      <Outlet />
    </>
  );
}
