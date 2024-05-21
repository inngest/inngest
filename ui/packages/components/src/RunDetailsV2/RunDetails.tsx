import { useCallback } from 'react';
import { toast } from 'sonner';

import { Trace } from '../TimelineV2';
import { Timeline } from '../TimelineV2/Timeline';
import { RunInfo } from './RunInfo';

type Props = {
  app: {
    name: string;
  };
  cancelRun: () => Promise<unknown>;
  fn: {
    name: string;
  };
  getOutput: (outputID: string) => Promise<string | null>;
  run: {
    id: string;
    output: string | null;
    trace: React.ComponentProps<typeof Trace>['trace'];
  };
};

export function RunDetails(props: Props) {
  const { app, getOutput, fn, run } = props;

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
      <RunInfo app={app} cancelRun={cancelRun} className="mb-4" fn={fn} run={run} />
      <Timeline getOutput={getOutput} trace={run.trace} />
    </div>
  );
}
