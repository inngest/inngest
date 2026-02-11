import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';
import { Switch } from '@inngest/components/Switch';
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
  const { value: v4Enabled } = booleanFlag('run-details-v4', false, true);
  const [showV4, setShowV4] = useState(true);
  const [runID] = useSearchParam('runID');
  const getTrigger = useGetTrigger();

  if (!runID) {
    throw new Error('missing runID in search params');
  }

  const pollInterval = pollingFlagReady && pollingDisabled ? 0 : 2500;

  // When flag is OFF, always show V3. When ON, use toggle state.
  const useV4 = v4Enabled && showV4;

  return (
    <div className={cn('bg-canvasBase overflow-y-auto pt-8')}>
      {/* Toggle switch for V3/V4 comparison - only visible when feature flag is enabled */}
      {v4Enabled && (
        <div className="mb-4 flex items-center gap-2 px-4">
          <span
            className={cn(
              'text-sm',
              !showV4 ? 'text-basis font-medium' : 'text-muted',
            )}
          >
            V3
          </span>
          <Switch checked={showV4} onCheckedChange={setShowV4} />
          <span
            className={cn(
              'text-sm',
              showV4 ? 'text-basis font-medium' : 'text-muted',
            )}
          >
            V4
          </span>
        </div>
      )}

      {useV4 ? (
        <RunDetailsV4
          standalone
          getTrigger={getTrigger}
          pollInterval={pollInterval}
          runID={runID}
        />
      ) : (
        <RunDetailsV3
          standalone
          getTrigger={getTrigger}
          pollInterval={pollInterval}
          runID={runID}
          newStack={true}
        />
      )}
    </div>
  );
}
