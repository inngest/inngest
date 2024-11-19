import type { Route } from 'next';

import { toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';
import { Trace } from './Trace';

type Props = {
  getResult: React.ComponentProps<typeof Trace>['getResult'];
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  trace: Lazy<React.ComponentProps<typeof Trace>['trace']>;
  stepAIEnabled?: boolean;
};

export function TimelineV2({ getResult, pathCreator, trace, stepAIEnabled = false }: Props) {
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
        trace={{ ...trace, childrenSpans: [], name: 'Run' }}
        stepAIEnabled={stepAIEnabled}
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
            stepAIEnabled={stepAIEnabled}
          />
        );
      })}
    </div>
  );
}
