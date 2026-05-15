import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
import { useTripleEscapeToggle } from '@inngest/components/hooks/useTripleEscapeToggle';
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
  const showV4 = useTripleEscapeToggle();

  // When flag is OFF, always show V3. When ON, use toggle state.
  const useV4 = v4Enabled && showV4;

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
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
