import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
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
  const [useV4, setUseV4] = useState(true);

  if (!runID) {
    throw new Error('missing runID in search params');
  }

  const pollInterval = pollingFlagReady && pollingDisabled ? 0 : 2500;

  return (
    <div className={cn('bg-canvasBase overflow-y-auto pt-8')}>
      {/* Toggle switch for V3/V4 comparison */}
      <div className="mb-4 flex items-center gap-2 px-4">
        <span
          className={cn(
            'text-sm',
            !useV4 ? 'text-basis font-medium' : 'text-muted',
          )}
        >
          V3
        </span>
        <button
          type="button"
          role="switch"
          aria-checked={useV4}
          onClick={() => setUseV4(!useV4)}
          className={cn(
            'relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-opacity-75',
            useV4 ? 'bg-primary-moderate' : 'bg-surfaceMuted',
          )}
        >
          <span
            aria-hidden="true"
            className={cn(
              'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out',
              useV4 ? 'translate-x-4' : 'translate-x-0',
            )}
          />
        </button>
        <span
          className={cn(
            'text-sm',
            useV4 ? 'text-basis font-medium' : 'text-muted',
          )}
        >
          V4
        </span>
      </div>

      {useV4 ? (
        <RunDetailsV4
          standalone
          getTrigger={getTrigger}
          pollInterval={pollInterval}
          runID={runID}
          newStack={true}
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
