import type { Trace as TraceType } from '../RunDetailsV3/types';
import { toMaybeDate } from '../utils/date';
import { DebugTrace } from './DebugTrace';

type Props = {
  runID: string;
  debugTraces?: TraceType[];
  runTrace: TraceType;
};

export const DebugRun = ({ debugTraces, runTrace, runID }: Props) => {
  if (!runID) {
    console.error('DebugRun component currently requires a runID', runID);
    return null;
  }

  //
  // hrm....what to do about total run time for a debug run where things may be
  // paused for long periods of time? for now using the original duration
  const minTime = new Date(runTrace.queuedAt);
  const maxTime = toMaybeDate(runTrace.endedAt) ?? new Date();

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
