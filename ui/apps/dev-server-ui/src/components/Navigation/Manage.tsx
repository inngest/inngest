import { useEffect, useState } from 'react';

import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';

import { useGetAppsQuery } from '@/store/generated';
import { MenuItem } from '@inngest/components/Menu/NewMenuItem';

export default function Mange({ collapsed }: { collapsed: boolean }) {
  const [pollingInterval, setPollingInterval] = useState(1500);
  const { booleanFlag } = useBooleanFlag();
  const { value: pollingDisabled, isReady: pollingFlagReady } = booleanFlag(
    'polling-disabled',
    false,
  );

  useEffect(() => {
    if (pollingFlagReady && pollingDisabled) {
      setPollingInterval(0);
    }
  }, [pollingDisabled, pollingFlagReady]);

  const { hasSyncingError } = useGetAppsQuery(undefined, {
    selectFromResult: (result) => ({
      hasSyncingError: result?.data?.apps?.some(
        (app) => app.connected === false,
      ),
    }),
    pollingInterval: pollingFlagReady && pollingDisabled ? 0 : pollingInterval,
  });

  return (
    <div className={`jusity-center mt-5 flex flex-col`}>
      {collapsed ? (
        <div className="border-subtle mx-auto mb-1 w-6 border-b" />
      ) : (
        <div className="text-muted leading-4.5 mb-1 text-xs font-medium">
          Manage
        </div>
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
