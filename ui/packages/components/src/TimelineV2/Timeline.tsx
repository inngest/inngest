import type { Route } from 'next';

import { toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';
import { Trace } from './Trace';

type Props = {
  getResult: React.ComponentProps<typeof Trace>['getResult'];
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  runID: string;
  trace: Lazy<React.ComponentProps<typeof Trace>['trace']>;
};

export function TimelineV2({ getResult, pathCreator, runID, trace }: Props) {
  if (!isLazyDone(trace)) {
    // TODO: Properly handle loading state
    return null;
  }

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
        runID={runID}
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
            runID={runID}
            trace={child}
          />
        );
      })}
    </div>
  );
}
