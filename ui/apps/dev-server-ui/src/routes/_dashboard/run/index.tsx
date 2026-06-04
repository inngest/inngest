import { createFileRoute } from '@tanstack/react-router';
import { RunDetailsV4 } from '@inngest/components/RunDetails';
import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';
import { useSearchParam } from '@inngest/components/hooks/useSearchParams';
import { cn } from '@inngest/components/utils/classNames';

import { useGetTrigger } from '@/hooks/useGetTrigger';

export const Route = createFileRoute('/_dashboard/run/')({
  component: RunComponent,
});

function RunComponent() {
  const { booleanFlag } = useBooleanFlag();
  const { value: pollingDisabled, isReady: pollingFlagReady } = booleanFlag(
    'polling-disabled',
    false,
  );

  const [runID] = useSearchParam('runID');
  const getTrigger = useGetTrigger();

  if (!runID) {
    throw new Error('missing runID in search params');
  }

  const pollInterval = pollingFlagReady && pollingDisabled ? 0 : 2500;

  return (
    <div className={cn('bg-canvasBase overflow-y-auto pt-8')}>
      <RunDetailsV4
        standalone
        getTrigger={getTrigger}
        pollInterval={pollInterval}
        runID={runID}
      />
    </div>
  );
}
