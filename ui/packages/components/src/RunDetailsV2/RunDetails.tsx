'use client';

import type { UrlObject } from 'url';
import { useCallback, useEffect, useState } from 'react';
import type { Route } from 'next';
import { toast } from 'sonner';

import { RunResult } from '../RunResult';
import { Trace } from '../TimelineV2';
import { Timeline } from '../TimelineV2/Timeline';
import type { Result } from '../types/functionRun';
import { RunInfo } from './RunInfo';

type Props = {
  standalone: boolean;
  app: {
    name: string;
    url: Route | UrlObject;
  };
  cancelRun: () => Promise<unknown>;
  fn: {
    id: string;
    name: string;
  };
  getResult: (outputID: string) => Promise<Result>;
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  rerun: (args: { fnID: string }) => Promise<unknown>;
  run: {
    id: string;
    trace: React.ComponentProps<typeof Trace>['trace'];
    url: Route | UrlObject;
  };
};

export function RunDetails(props: Props) {
  const { app, getResult, fn, pathCreator, rerun, run, standalone } = props;
  const [result, setResult] = useState<Result>();

  useEffect(() => {
    if (!result && run.trace.outputID) {
      getResult(run.trace.outputID).then((data) => {
        setResult(data);
      });
    }
  }, [result, run.trace.outputID]);

  const cancelRun = useCallback(async () => {
    try {
      await props.cancelRun();
      toast.success('Cancelled run');
    } catch (e) {
      toast.error('Failed to cancel run');
      console.error(e);
    }
  }, [props.cancelRun]);

  return (
    <div className="pr-4">
      <RunInfo
        app={app}
        cancelRun={cancelRun}
        className="mb-4"
        fn={fn}
        rerun={rerun}
        run={run}
        standalone={standalone}
      />

      {result && <RunResult className="mb-4" result={result} />}

      <Timeline getResult={getResult} pathCreator={pathCreator} trace={run.trace} />
    </div>
  );
}
