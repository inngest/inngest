'use client';

import { useCallback, useEffect, useState } from 'react';
import type { Route } from 'next';
import { toast } from 'sonner';

import { StatusCell } from '../Table';
import { Trace } from '../TimelineV2';
import { Timeline } from '../TimelineV2/Timeline';
import { TriggerDetails } from '../TriggerDetails';
import type { Result } from '../types/functionRun';
import { nullishToLazy } from '../utils/lazyLoad';
import { withRetry } from '../utils/retry';
import { RunInfo } from './RunInfo';

type Props = {
  standalone: boolean;
  cancelRun: (runID: string) => Promise<unknown>;
  getResult: (outputID: string) => Promise<Result>;
  getRun: (runID: string) => Promise<Run>;
  getTrigger: React.ComponentProps<typeof TriggerDetails>['getTrigger'];
  pathCreator: {
    app: (params: { externalAppID: string }) => Route;
    runPopout: (params: { runID: string }) => Route;
  };
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
  };
  id: string;
  trace: React.ComponentProps<typeof Trace>['trace'];
};

export function RunDetails(props: Props) {
  const { getResult, getRun, getTrigger, pathCreator, rerun, runID, standalone } = props;
  const [error, setError] = useState<Error>();

  const [run, setRun] = useState<Run>();
  useEffect(() => {
    if (!run) {
      withRetry(() => getRun(runID))
        .then((data) => {
          setRun(data);
        })
        .catch(setError);
    }
  }, []);

  const [result, setResult] = useState<Result>();
  const outputID = run?.trace?.outputID;
  useEffect(() => {
    if (!result && outputID) {
      withRetry(() => getResult(outputID))
        .then((data) => {
          setResult(data);
        })
        .catch(() => {
          toast.error('Failed to fetch run result');
        });
    }
  }, [result, outputID]);

  const cancelRun = useCallback(async () => {
    try {
      await props.cancelRun(runID);
      toast.success('Cancelled run');
    } catch (e) {
      toast.error('Failed to cancel run');
      console.error(e);
    }
  }, [props.cancelRun]);

  if (error) {
    throw error;
    // return (
    //   <div className="m-auto mt-32 flex w-fit flex-col gap-4">
    //     <Alert
    //       severity="error"
    //       button={
    //         <NewButton
    //           onClick={() => window.location.reload()}
    //           kind="secondary"
    //           appearance="outlined"
    //           label="Refresh page"
    //         />
    //       }
    //     >
    //       <p className="mb-4 font-semibold">{error.message}</p>

    //       <p>An error occurred! Refresh the page to try again.</p>
    //     </Alert>
    //   </div>
    // );
  }

  return (
    <div>
      {standalone && run && (
        <div className="mx-8 flex flex-col gap-1 pb-6">
          <StatusCell status={run.trace.status} />
          <p className="text-2xl font-medium">{run.fn.name}</p>
          <p className="font-mono text-slate-500">{runID}</p>
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
              run={nullishToLazy(run)}
              runID={runID}
              standalone={standalone}
              result={result}
            />
          </div>

          {run && <Timeline getResult={getResult} pathCreator={pathCreator} trace={run.trace} />}
        </div>

        <TriggerDetails getTrigger={getTrigger} runID={runID} />
      </div>
    </div>
  );
}
