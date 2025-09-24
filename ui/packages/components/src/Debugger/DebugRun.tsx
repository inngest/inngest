import type { Trace as TraceType } from '../RunDetailsV3/types';
import { toMaybeDate } from '../utils/date';
import { DebugTrace } from './DebugTrace';

type Props = {
  runID?: string;
  debugTraces?: TraceType[];
  runTrace?: TraceType;
};

export const DebugRun = ({ debugTraces, runTrace, runID }: Props) => {
  if (!runID) {
    console.error('DebugRun component currently requires a runID', runID);
    return null;
  }

  if (!runTrace) {
    console.error('DebugRun component currently requires a runTrace', runID);
    return null;
  }

  const latest = debugTraces?.at(-1);
  const minTime = latest ? new Date(latest.queuedAt) : new Date(runTrace.queuedAt);
  const maxTime = latest?.endedAt
    ? new Date(latest.endedAt)
    : toMaybeDate(runTrace.endedAt) ?? new Date();

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
