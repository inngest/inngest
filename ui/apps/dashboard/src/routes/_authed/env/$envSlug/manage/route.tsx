import { createFileRoute, Outlet } from '@tanstack/react-router';

import { Route as OrgActiveRoute } from '@/routes/_authed';
import ChildEmptyState from '@/components/Manage/ChildEmptyState';
import { ManageHeader } from '@/components/Manage/Header';

export const Route = createFileRoute('/_authed/env/$envSlug/manage')({
  component: ManageLayoutComponent,
});

function ManageLayoutComponent() {
  const { env } = OrgActiveRoute?.useLoaderData() ?? {};

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
