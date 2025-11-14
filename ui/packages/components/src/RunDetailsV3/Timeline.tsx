import { max, min } from 'date-fns';

import { toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';
import { Trace } from './Trace';
import { traceWalk } from './utils';

type Props = {
  runID: string;
  trace: Lazy<React.ComponentProps<typeof Trace>['trace']>;
};

export const Timeline = ({ runID, trace }: Props) => {
  if (!isLazyDone(trace)) {
    // TODO: Properly handle loading state
    return null;
  }

  let minTime = new Date(trace.queuedAt);
  let maxTime = toMaybeDate(trace.endedAt) ?? new Date();

  traceWalk(trace, (t) => {
    minTime = min([minTime, new Date(t.queuedAt)]);

    const endedAt = toMaybeDate(t.endedAt);
    if (endedAt) {
      maxTime = max([endedAt, maxTime]);
    }
  });

  return (
    <div className={`w-full pb-4 pr-8`}>
      <Trace
        depth={0}
        maxTime={maxTime}
        minTime={minTime}
        runID={runID}
        trace={{ ...trace, name: 'Run' }}
      />
    </div>
  );
};
