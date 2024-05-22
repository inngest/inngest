import { toMaybeDate } from '../utils/date';
import { Trace } from './Trace';

type Props = {
  getOutput: (outputID: string) => Promise<string | null>;
  pathCreator: {
    runPopout: (params: { runID: string }) => string;
  };
  trace: React.ComponentProps<typeof Trace>['trace'];
};

export function Timeline({ getOutput, pathCreator, trace }: Props) {
  const minTime = new Date(trace.queuedAt);
  const maxTime = toMaybeDate(trace.endedAt) ?? new Date();

  return (
    <div>
      <Trace
        depth={0}
        getOutput={getOutput}
        isExpandable={false}
        maxTime={maxTime}
        minTime={minTime}
        pathCreator={pathCreator}
        trace={{ ...trace, childrenSpans: [], name: 'Run' }}
      />

      {trace.childrenSpans?.map((child) => {
        return (
          <Trace
            depth={0}
            getOutput={getOutput}
            key={child.spanID}
            maxTime={maxTime}
            minTime={minTime}
            pathCreator={pathCreator}
            trace={child}
          />
        );
      })}
    </div>
  );
}
