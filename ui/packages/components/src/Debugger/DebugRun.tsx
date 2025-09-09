import { Trace } from '../RunDetailsV3/Trace';
import type { RunTraceSpan } from '../SharedContext/useGetDebugRun';
import { toMaybeDate } from '../utils/date';

type Props = {
  debugRun: RunTraceSpan;
};

export const DebugRun = ({ debugRun }: Props) => {
  const minTime = new Date(debugRun.queuedAt);
  const maxTime = toMaybeDate(debugRun.endedAt) ?? new Date();

  return (
    <div className={`w-full pb-4 pr-8`}>
      <Trace
        depth={0}
        maxTime={maxTime}
        minTime={minTime}
        runID={debugRun.runID}
        trace={{ ...(debugRun as any), name: 'Debug Run' }}
      />
    </div>
  );
};
