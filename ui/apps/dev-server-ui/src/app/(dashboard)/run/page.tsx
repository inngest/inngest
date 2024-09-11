'use client';

import { RunDetails } from '@inngest/components/RunDetailsV2/RunDetails';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { cn } from '@inngest/components/utils/classNames';

import { useCancelRun } from '@/hooks/useCancelRun';
import { useGetRun } from '@/hooks/useGetRun';
import { useGetTraceResult } from '@/hooks/useGetTraceResult';
import { useGetTrigger } from '@/hooks/useGetTrigger';
import { useRerun } from '@/hooks/useRerun';
import { pathCreator } from '@/utils/pathCreator';

export default function Page() {
  const [runID] = useSearchParam('runID');
  const cancelRun = useCancelRun();
  const getRun = useGetRun();
  const getTraceResult = useGetTraceResult();
  const getTrigger = useGetTrigger();
  const rerun = useRerun();

  if (!runID) {
    throw new Error('missing runID in search params');
  }

  return (
    <div className={cn('bg-canvasBase overflow-y-auto pt-8')}>
      <RunDetails
        pathCreator={pathCreator}
        standalone
        cancelRun={cancelRun}
        getResult={getTraceResult}
        getRun={getRun}
        getTrigger={getTrigger}
        pollInterval={2500}
        rerun={rerun}
        runID={runID}
        // TODO - initialRunData={{}}
      />
    </div>
  );
}
