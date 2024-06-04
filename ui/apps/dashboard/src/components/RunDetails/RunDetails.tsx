'use client';

import { useMemo } from 'react';
import { RunDetails as RunDetailsView } from '@inngest/components/RunDetailsV2';
import { StatusCell } from '@inngest/components/Table';
import { TriggerDetails } from '@inngest/components/TriggerDetails';
import { cn } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { toRunStatus } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/runs/utils';
import { useCancelRun } from '@/queries/useCancelRun';
import { useRerun } from '@/queries/useRerun';
import { pathCreator } from '@/utils/urls';
import { useGetTraceResult } from './useGetTraceResult';
import { useGetTrigger } from './useGetTrigger';
import { useRun } from './useRun';

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
      runPopout: (params: { runID: string }) =>
        pathCreator.runPopout({ envSlug: env.slug, runID: params.runID }),
    };
  }, [env.slug]);

  const getTrigger = useGetTrigger({ runID });

  const res = useRun({ envID: env.id, runID });
  if (res.error) {
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return <Loading />;
  }
  const { run, trace } = res.data;

  return (
    <div className={cn('overflow-y-auto', standalone && 'p-5 pt-8')}>
      {standalone && (
        <div className="flex flex-col gap-2 pb-6">
          {/* @ts-ignore */}
          {run.trace && <StatusCell status={toRunStatus(run.trace.status)} />}
          <p className="text-2xl font-medium">{run.function.name}</p>
          <p className="font-mono text-slate-500">{runID}</p>
        </div>
      )}
      <div className="flex">
        <div className="flex-1">
          <RunDetailsView
            pathCreator={internalPathCreator}
            standalone={standalone}
            app={{
              url: pathCreator.app({
                envSlug: env.slug,
                externalAppID: run.function.app.externalID,
              }),
              ...run.function.app,
            }}
            cancelRun={cancelRun}
            fn={run.function}
            getResult={getTraceResult}
            rerun={rerun}
            run={{
              id: runID,
              trace,
              url: pathCreator.runPopout({ envSlug: env.slug, runID: runID }),
            }}
          />
        </div>
        <TriggerDetails getTrigger={getTrigger} />
      </div>
    </div>
  );
}

function Loading() {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <div className="flex flex-col items-center justify-center gap-2">
        <div>Loading</div>
      </div>
    </div>
  );
}
