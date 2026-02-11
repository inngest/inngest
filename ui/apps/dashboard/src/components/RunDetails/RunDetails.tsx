import { useState } from 'react';
import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
import { Switch } from '@inngest/components/Switch';
import { cn } from '@inngest/components/utils/classNames';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useGetTrigger } from './useGetTrigger';

type Props = {
  runID: string;
  standalone?: boolean;
};

export const DEFAULT_POLL_INTERVAL = 1000;

export function DashboardRunDetails({ runID, standalone = true }: Props) {
  const getTrigger = useGetTrigger();
  const { value: v4Enabled } = useBooleanFlag('run-details-v4');
  const [showV4, setShowV4] = useState(true);

  // When flag is OFF, always show V3. When ON, use toggle state.
  const useV4 = v4Enabled && showV4;

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
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
          standalone={standalone}
          getTrigger={getTrigger}
          runID={runID}
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
