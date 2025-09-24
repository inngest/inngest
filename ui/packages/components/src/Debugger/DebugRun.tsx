import type { Trace as TraceType } from '../RunDetailsV3/types';
import { toMaybeDate } from '../utils/date';
import { DebugTrace } from './DebugTrace';

type Props = {
  runID: string;
  debugTraces: TraceType[];
  runTrace: TraceType;
};

export const DebugRun = ({ debugTraces, runTrace, runID }: Props) => {
  const latest = debugTraces?.at(-1);

  if (!latest) {
    return null;
  }

  const minTime = new Date(latest.queuedAt);
  const maxTime = toMaybeDate(latest.endedAt) ?? new Date();

  return (
    <div className={`w-full pb-4 pr-8`}>
      <DebugTrace
        depth={0}
        maxTime={maxTime}
        minTime={minTime}
        runID={runID}
        runTrace={runTrace}
        debugTraces={debugTraces}
      />
    </div>
  );
};
