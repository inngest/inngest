import { Outlet, createFileRoute } from '@tanstack/react-router';

// import SideBar from '@/components/Layout/SideBar';
import { EnvironmentType } from '@/utils/environments';

export const Route = createFileRoute('/env')({
  component: Component,
});

function Component() {
  return (
    <div>
      {/* <SideBar
        activeEnv={{
          type: EnvironmentType.Production,
          id: '123',
          hasParent: true,
          name: 'Production',
          slug: 'production',
          webhookSigningKey: '123',
          createdAt: '2024-01-01',
          isArchived: false,
          isAutoArchiveEnabled: false,
          lastDeployedAt: '2024-01-01',
        }}
        collapsed={false}
        enableQuickSearchV2
        profile={{
          isMarketplace: false,
          orgName: 'Org',
          displayName: 'John Doe',
        }}
      /> */}
      yo
      <Outlet />
    </div>
  );
}
