import { Trace } from './Trace';
import { toMaybeDate } from './utils';

type Props = {
  getOutput: (outputID: string) => Promise<string | null>;
  trace: React.ComponentProps<typeof Trace>['trace'];
};

export function Timeline({ getOutput, trace }: Props) {
  const minTime = new Date(trace.queuedAt);
  const maxTime = toMaybeDate(trace.endedAt) ?? new Date();

  return (
    <div className="border-t border-slate-300">
      <Trace
        depth={0}
        getOutput={getOutput}
        isExpandable={false}
        maxTime={maxTime}
        minTime={minTime}
        trace={{ ...trace, childrenSpans: [], name: 'Run' }}
      />

      {trace.childrenSpans?.map((child) => {
        return (
          <Trace
            depth={0}
            getOutput={getOutput}
            key={child.id}
            maxTime={maxTime}
            minTime={minTime}
            trace={child}
          />
        );
      })}
    </div>
  );
}
