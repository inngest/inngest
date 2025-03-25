'use client';

import { useMemo } from 'react';
import { RunDetailsV2 } from '@inngest/components/RunDetailsV2';
import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3';
import { useLegacyTrace } from '@inngest/components/SharedContext/useLegacyTrace';
import { cn } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/components/Environments/environment-context';
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
  const getTraceResult = useGetTraceResult();
  const { value: traceAIEnabled, isReady } = useBooleanFlag('ai-traces');

  const { enabled: legacyTraceEnabled } = useLegacyTrace();

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
      {isReady && traceAIEnabled && !legacyTraceEnabled ? (
        <RunDetailsV3
          pathCreator={internalPathCreator}
          standalone={standalone}
          getResult={getTraceResult}
          getRun={getRun}
          getTrigger={getTrigger}
          runID={runID}
        />
      ) : (
        <RunDetailsV2
          pathCreator={internalPathCreator}
          standalone={standalone}
          getResult={getTraceResult}
          getRun={getRun}
          getTrigger={getTrigger}
          runID={runID}
          traceAIEnabled={traceAIEnabled}
        />
      )}
    </div>
  );
}
