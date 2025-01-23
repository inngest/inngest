'use client';

import { useCallback, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import { ErrorCard } from '../RunDetailsV2/ErrorCard';
import type { Run as InitialRunData } from '../RunsPage/types';
import { Trace as OldTrace } from '../TimelineV2';
import { TriggerDetails } from '../TriggerDetails';
import type { Result } from '../types/functionRun';
import { nullishToLazy } from '../utils/lazyLoad';
import { RunInfo } from './RunInfo';
import { Trace } from './Trace';
import { TriggerInfo } from './TriggerInfo';

type Props = {
  standalone: boolean;
  cancelRun: (runID: string) => Promise<unknown>;
  getResult: (outputID: string) => Promise<Result>;
  getRun: (runID: string) => Promise<Run>;
  initialRunData?: InitialRunData;
  getTrigger: React.ComponentProps<typeof TriggerDetails>['getTrigger'];
  pathCreator: React.ComponentProps<typeof RunInfo>['pathCreator'];
  pollInterval?: number;
  rerun: React.ComponentProps<typeof RunInfo>['rerun'];
  rerunFromStep: React.ComponentProps<typeof RunInfo>['rerunFromStep'];
  runID: string;
};

type Run = {
  app: {
    externalID: string;
    name: string;
  };
  fn: {
    id: string;
    name: string;
    slug: string;
  };
  id: string;
  trace: React.ComponentProps<typeof OldTrace>['trace'];
  hasAI: boolean;
};

export const RunDetailsV3 = (props: Props) => {
  const { getResult, getRun, getTrigger, pathCreator, rerun, rerunFromStep, runID, standalone } =
    props;
  const [pollInterval, setPollInterval] = useState(props.pollInterval);

  const runRes = useQuery({
    queryKey: ['run', runID],
    queryFn: useCallback(() => {
      return getRun(runID);
    }, [getRun, runID]),
    retry: 3,
    refetchInterval: pollInterval,
  });

  const outputID = runRes?.data?.trace.outputID;
  const resultRes = useQuery({
    enabled: Boolean(outputID),
    queryKey: ['run-result', runID],
    queryFn: useCallback(() => {
      if (!outputID) {
        // Unreachable
        throw new Error('missing outputID');
      }

      return getResult(outputID);
    }, [getResult, outputID]),
  });

  const cancelRun = useCallback(async () => {
    return await props.cancelRun(runID);
  }, [props.cancelRun]);

  const run = runRes.data;
  if (run?.trace.endedAt && pollInterval) {
    // Stop polling since ended runs are immutable
    setPollInterval(undefined);
  }

  // Do not show the error if queued and the error is no spans
  const isNoSpansFoundError = !!runRes.error?.toString().match(/no function run span found/gi);
  const showError =
    props.initialRunData?.status === 'QUEUED' && isNoSpansFoundError
      ? false
      : runRes.error || resultRes.error;

  return (
    <div className="ml-4 mt-4 flex h-full flex-row">
      <div className="flex h-full w-[55%] flex-col">
        <div className="h-full resize-x pb-4 pr-4">
          <RunInfo
            cancelRun={cancelRun}
            className="mb-4"
            pathCreator={pathCreator}
            rerun={rerun}
            rerunFromStep={rerunFromStep}
            initialRunData={props.initialRunData}
            run={nullishToLazy(run)}
            runID={runID}
            standalone={standalone}
            result={resultRes.data}
            traceAIEnabled={false}
          />
          {showError && (
            <ErrorCard
              error={runRes.error || resultRes.error}
              reset={runRes.error ? () => runRes.refetch() : () => resultRes.refetch()}
            />
          )}
        </div>
        <Trace />
      </div>
      <div className="border-muted flex h-full w-[45%] resize-x flex-col border-l">
        <TriggerInfo getTrigger={getTrigger} runID={runID} />
      </div>
    </div>
  );
};
