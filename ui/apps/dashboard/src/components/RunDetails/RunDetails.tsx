import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
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

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
      {v4Enabled ? (
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
