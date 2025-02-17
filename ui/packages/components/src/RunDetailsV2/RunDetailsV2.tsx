'use client';

import { useCallback, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import type { Run as InitialRunData } from '../RunsPage/types';
import { StatusCell } from '../Table';
import { Trace as OldTrace } from '../TimelineV2';
import { TimelineV2 } from '../TimelineV2/Timeline';
import { TriggerDetails } from '../TriggerDetails';
import type { Result } from '../types/functionRun';
import { nullishToLazy } from '../utils/lazyLoad';
import { ErrorCard } from './ErrorCard';
import { RunInfo } from './RunInfo';

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

export function RunDetailsV2(props: Props) {
  const { getResult, getRun, getTrigger, pathCreator, rerun, runID, standalone } = props;
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
    <div>
      {standalone && run && (
        <div className="mx-8 flex flex-col gap-1 pb-6">
          <StatusCell status={run.trace.status} />
          <p className="text-basis text-2xl font-medium">{run.fn.name}</p>
          <p className="text-subtle font-mono">{runID}</p>
        </div>
      )}

      <div className="flex gap-4">
        <div className="grow">
          <div className="ml-8">
            <RunInfo
              cancelRun={cancelRun}
              className="mb-4"
              pathCreator={pathCreator}
              rerun={rerun}
              initialRunData={props.initialRunData}
              run={nullishToLazy(run)}
              runID={runID}
              standalone={standalone}
              result={resultRes.data}
            />
            {showError && (
              <ErrorCard
                error={runRes.error || resultRes.error}
                reset={runRes.error ? () => runRes.refetch() : () => resultRes.refetch()}
              />
            )}
          </div>

          {run && (
            <TimelineV2
              getResult={getResult}
              pathCreator={pathCreator}
              runID={runID}
              trace={run.trace}
            />
          )}
        </div>

        <TriggerDetails getTrigger={getTrigger} runID={runID} />
      </div>
    </div>
  );
}
