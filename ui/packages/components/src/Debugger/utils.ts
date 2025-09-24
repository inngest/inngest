import type { Trace } from '../RunDetailsV3/types';

/**
 * TODO: this can go away when our individual debug traces are complete. Currently
 * they are partial an only contain the step from which the were run onward.
 *
 * This mapper papers over that ^
 */
export const debugTraceMapper = (runTrace: Trace, debugTraces: Trace[]) => {
  return debugTraces.map((trace) => {
    return {
      ...trace,
      debugRunID: trace.debugRunID,
      debugSessionID: trace.debugSessionID,
    };
  });
};
