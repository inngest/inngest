import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';

import { useGetAppsQuery } from '@/store/generated';

export default function Mange({ collapsed }: { collapsed: boolean }) {
  const { hasSyncingError } = useGetAppsQuery(undefined, {
    selectFromResult: (result) => ({
      hasSyncingError: result?.data?.apps?.some((app) => app.connected === false),
    }),
    pollingInterval: 1500,
  });

  return (
    <div className={`jusity-center mt-5 flex flex-col`}>
      {collapsed ? (
        <div className="border-subtle mx-auto mb-1 w-6 border-b" />
      ) : (
        <div className="text-muted leading-4.5 mx-2.5 mb-1 text-xs font-medium">Manage</div>
      )}
      <MenuItem
        href="/apps"
        collapsed={collapsed}
        text="Apps"
        icon={<AppsIcon className="h-[18px] w-[18px]" />}
        error={hasSyncingError}
      />

      <MenuItem
        href="/functions"
        collapsed={collapsed}
        text="Functions"
        icon={<FunctionsIcon className="h-[18px] w-[18px]" />}
      />
    </div>
  );
}
