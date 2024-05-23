import type { UrlObject } from 'url';
import { useCallback } from 'react';
import type { Route } from 'next';
import { toast } from 'sonner';

import { Trace } from '../TimelineV2';
import { Timeline } from '../TimelineV2/Timeline';
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
  getOutput: (outputID: string) => Promise<string | null>;
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  rerun: (args: { fnID: string }) => Promise<unknown>;
  run: {
    id: string;
    output: string | null;
    trace: React.ComponentProps<typeof Trace>['trace'];
    url: Route | UrlObject;
  };
};

export function RunDetails(props: Props) {
  const { app, getOutput, fn, pathCreator, rerun, run, standalone } = props;

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
    <div>
      <RunInfo
        app={app}
        cancelRun={cancelRun}
        className="mb-4"
        fn={fn}
        rerun={rerun}
        run={run}
        standalone={standalone}
      />
      <Timeline getOutput={getOutput} pathCreator={pathCreator} trace={run.trace} />
    </div>
  );
}
