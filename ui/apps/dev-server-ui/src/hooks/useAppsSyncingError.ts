import { useEffect, useState } from 'react';

import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';

import { useGetAppsQuery } from '@/store/generated';

// Polls the apps list and reports whether any app has failed to sync, so the
// nav can surface an error badge on the Apps item.
export const useAppsSyncingError = (): boolean | undefined => {
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

  return hasSyncingError;
};
