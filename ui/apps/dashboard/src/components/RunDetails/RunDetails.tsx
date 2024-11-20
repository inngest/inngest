'use client';

import { useMemo } from 'react';
import { RunDetailsV2 } from '@inngest/components/RunDetailsV2';
import { cn } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useCancelRun } from '@/queries/useCancelRun';
import { useRerun } from '@/queries/useRerun';
import { useRerunFromStep } from '@/queries/useRerunFromStep';
import { pathCreator } from '@/utils/urls';
import { useBooleanFlag } from '../FeatureFlags/hooks';
import { useGetRun } from './useGetRun';
import { useGetTraceResult } from './useGetTraceResult';
import { useGetTrigger } from './useGetTrigger';

type Props = {
  runID: string;
  standalone?: boolean;
};

export function DashboardRunDetails({ runID, standalone = true }: Props) {
  const env = useEnvironment();
  const cancelRun = useCancelRun({ envID: env.id });
  const rerun = useRerun({ envID: env.id, envSlug: env.slug });
  const rerunFromStep = useRerunFromStep({ runID, fromStep: { stepID: 'stepID', input: 'input' } });
  const getTraceResult = useGetTraceResult();
  const { value: stepAIEnabled, isReady } = useBooleanFlag('step.ai');

  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      app: (params: { externalAppID: string }) =>
        pathCreator.app({ envSlug: env.slug, externalAppID: params.externalAppID }),
      function: (params: { functionSlug: string }) =>
        pathCreator.function({ envSlug: env.slug, functionSlug: params.functionSlug }),
      runPopout: (params: { runID: string }) =>
        pathCreator.runPopout({ envSlug: env.slug, runID: params.runID }),
    };
  }, [env.slug]);

  const getTrigger = useGetTrigger();
  const getRun = useGetRun();

  return (
    <div className={cn('overflow-y-auto', standalone && 'pt-8')}>
      <RunDetailsV2
        pathCreator={internalPathCreator}
        standalone={standalone}
        cancelRun={cancelRun}
        getResult={getTraceResult}
        getRun={getRun}
        getTrigger={getTrigger}
        rerun={rerun}
        rerunFromStep={rerunFromStep}
        runID={runID}
        stepAIEnabled={isReady && stepAIEnabled}
      />
    </div>
  );
}
