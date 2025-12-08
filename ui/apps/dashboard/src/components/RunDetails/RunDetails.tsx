import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { cn } from '@inngest/components/utils/classNames';

import { useGetTrigger } from './useGetTrigger';

type Props = {
  runID: string;
  standalone?: boolean;
};

export const DEFAULT_POLL_INTERVAL = 1000;

export function DashboardRunDetails({ runID, standalone = true }: Props) {
  const getTrigger = useGetTrigger();

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
      <RunDetailsV3
        standalone={standalone}
        getTrigger={getTrigger}
        runID={runID}
        newStack={true}
        pollInterval={DEFAULT_POLL_INTERVAL}
      />
    </div>
  );
}
