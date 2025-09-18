import { Trace } from '../RunDetailsV3/Trace';
import type { Trace as TraceType } from '../RunDetailsV3/types';
import { toMaybeDate } from '../utils/date';

type Props = {
  runID: string;
  debugRun: TraceType;
};

export const DebugRun = ({ debugRun, runID }: Props) => {
  const minTime = new Date(debugRun.queuedAt);
  const maxTime = toMaybeDate(debugRun.endedAt) ?? new Date();

  return (
    <div className={`w-full pb-4 pr-8`}>
      <Trace
        depth={0}
        maxTime={maxTime}
        minTime={minTime}
        runID={runID}
        trace={{ ...(debugRun as any), name: 'Debug Run' }}
      />
    </div>
  );
};
