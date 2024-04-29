import { Trace } from '../TimelineV2';
import { Timeline } from '../TimelineV2/Timeline';
import { RunInfo } from './RunInfo';

type Props = {
  app: {
    name: string;
  };
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

export function RunDetails({ app, getOutput, fn, run }: Props) {
  return (
    <div>
      <RunInfo app={app} className="mb-4" fn={fn} run={run} />
      <Timeline getOutput={getOutput} trace={run.trace} />
    </div>
  );
}
