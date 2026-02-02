import { useState } from 'react';
import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
import { cn } from '@inngest/components/utils/classNames';

import { useGetTrigger } from './useGetTrigger';

type Props = {
  runID: string;
  standalone?: boolean;
};

export const DEFAULT_POLL_INTERVAL = 1000;

export function DashboardRunDetails({ runID, standalone = true }: Props) {
  const getTrigger = useGetTrigger();
  const [useV4, setUseV4] = useState(true);

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
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
          standalone={standalone}
          getTrigger={getTrigger}
          runID={runID}
          newStack={true}
          pollInterval={DEFAULT_POLL_INTERVAL}
        />
      ) : (
        <RunDetailsV3
          standalone={standalone}
          getTrigger={getTrigger}
          runID={runID}
          newStack={true}
          pollInterval={DEFAULT_POLL_INTERVAL}
        />
      )}
    </div>
  );
}
