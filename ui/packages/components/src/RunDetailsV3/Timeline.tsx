import type { Route } from 'next';

import { toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';
import { Trace } from './Trace';

type Props = {
  pathCreator: {
    runPopout: (params: { runID: string }) => Route;
  };
  runID: string;
  trace: Lazy<React.ComponentProps<typeof Trace>['trace']>;
};

export const Timeline = ({ pathCreator, runID, trace }: Props) => {
  if (!isLazyDone(trace)) {
    // TODO: Properly handle loading state
    return null;
  }

  const minTime = new Date(trace.queuedAt);
  const maxTime = toMaybeDate(trace.endedAt) ?? new Date();

  return (
    <div className={`w-full pb-4 pr-8`}>
      <Trace
        depth={0}
        maxTime={maxTime}
        minTime={minTime}
        pathCreator={pathCreator}
        runID={runID}
        trace={{ ...trace, name: 'Run' }}
      />
    </div>
  );
};
