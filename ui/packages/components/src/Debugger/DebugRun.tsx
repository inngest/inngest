import { Trace } from '../RunDetailsV3/Trace';
import type { Trace as TraceType } from '../RunDetailsV3/types';
import { toMaybeDate } from '../utils/date';

type Props = {
  runID: string;
  debugRun: TraceType[];
};

export const DebugRun = ({ debugRun, runID }: Props) => {
  const latest = debugRun?.at(-1);

  if (!latest) {
    return null;
  }

  const minTime = new Date(latest.queuedAt);
  const maxTime = toMaybeDate(latest.endedAt) ?? new Date();

  return (
    <div className={`w-full pb-4 pr-8`}>
      <Trace
        depth={0}
        maxTime={maxTime}
        minTime={minTime}
        runID={runID}
        trace={{ ...(latest as any), name: 'Debug Run' }}
      />
    </div>
  );
};
