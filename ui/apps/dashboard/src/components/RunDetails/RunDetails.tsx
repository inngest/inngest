'use client';

import { useMemo } from 'react';
import { RunDetails as RunDetailsView } from '@inngest/components/RunDetailsV2';
import { cn } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { useCancelRun } from '@/queries/useCancelRun';
import { useRerun } from '@/queries/useRerun';
import { pathCreator } from '@/utils/urls';
import { useGetRun } from './useGetRun';
import { useGetTraceResult } from './useGetTraceResult';
import { useGetTrigger } from './useGetTrigger';

type Props = {
  runID: string;
  standalone?: boolean;
};

export function RunDetails({ runID, standalone = true }: Props) {
  const env = useEnvironment();
  const cancelRun = useCancelRun({ envID: env.id, runID });
  const rerun = useRerun({ envID: env.id, envSlug: env.slug, runID });
  const getTraceResult = useGetTraceResult();

  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      app: (params: { externalAppID: string }) =>
        pathCreator.app({ envSlug: env.slug, externalAppID: params.externalAppID }),
      runPopout: (params: { runID: string }) =>
        pathCreator.runPopout({ envSlug: env.slug, runID: params.runID }),
    };
  }, [env.slug]);

  const getTrigger = useGetTrigger({ runID });
  const getRun = useGetRun();

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
      <RunDetailsView
        pathCreator={internalPathCreator}
        standalone={standalone}
        cancelRun={cancelRun}
        getResult={getTraceResult}
        getRun={getRun}
        getTrigger={getTrigger}
        rerun={rerun}
        runID={runID}
      />
    </div>
  );
}
