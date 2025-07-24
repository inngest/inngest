'use client';

import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { cn } from '@inngest/components/utils/classNames';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useGetRun } from './useGetRun';
import { useGetTraceResult } from './useGetTraceResult';
import { useGetTrigger } from './useGetTrigger';

type Props = {
  runID: string;
  standalone?: boolean;
};

export function DashboardRunDetails({ runID, standalone = true }: Props) {
  const getTraceResult = useGetTraceResult();

  const getTrigger = useGetTrigger();
  const getRun = useGetRun();
  const { value: tracePreviewEnabled } = useBooleanFlag('traces-preview', false);

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
      <RunDetailsV3
        standalone={standalone}
        getResult={getTraceResult}
        getRun={getRun}
        getTrigger={getTrigger}
        runID={runID}
        tracesPreviewEnabled={tracePreviewEnabled}
      />
    </div>
  );
}
