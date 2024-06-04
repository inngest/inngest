import type { Route } from 'next';

import { toMaybeDate } from '../utils/date';
import { Trace } from './Trace';

type Props = {
  getResult: React.ComponentProps<typeof Trace>['getResult'];
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  trace: React.ComponentProps<typeof Trace>['trace'];
};

export function Timeline({ getResult, pathCreator, trace }: Props) {
  const minTime = new Date(trace.queuedAt);
  const maxTime = toMaybeDate(trace.endedAt) ?? new Date();

  return (
    <div>
      <Trace
        depth={0}
        getResult={getResult}
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
            getResult={getResult}
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
