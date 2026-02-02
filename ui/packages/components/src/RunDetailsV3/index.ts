/**
 * RunDetailsV3 module exports
 * Feature: EXE-1217 - step.run Timing Breakdown
 */

// Types
export type {
  Trace,
  TimingCategory,
  TimingSegment,
  TimingCategoryTotal,
  SpanTimingBreakdown,
  SegmentType,
  InngestSegmentType,
  ConnectingSegmentType,
  CustomerServerSegmentType,
} from './types';

// Type guards from types.ts
export {
  isStepInfoRun,
  isStepInfoInvoke,
  isStepInfoSleep,
  isStepInfoWait,
  isStepInfoSignal,
} from './types';

// Timing breakdown utilities
export {
  TIMING_COLORS,
  CATEGORY_ICONS,
  isStepRunSpan,
  formatDuration,
  calculateTimingBreakdown,
} from './timingBreakdown';

// Timing breakdown components
export { SegmentedSpanBar } from './SegmentedSpanBar';
export { SegmentedTimelineSpan } from './SegmentedTimelineSpan';
export { TimingBreakdownPanel } from './TimingBreakdownPanel';
