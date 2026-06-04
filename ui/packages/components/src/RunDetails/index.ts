/**
 * RunDetailsV4 - Composable Timeline Bar Component
 * Feature: 001-composable-timeline-bar
 */

// Types
export type {
  TimelineBarProps,
  BarStyleKey,
  BarStyle,
  BarSegment,
  BarPattern,
  BarIcon,
  TimelineData,
  TimelineBarData,
  TimingBreakdownData,
} from './TimelineBar.types';

// Components
export { TimelineBar } from './TimelineBar';
export { TimelineHeader } from './TimelineHeader';
export { Timeline } from './Timeline';
export { RunDetailsV4 } from './RunDetailsV4';

// Utilities
export { formatDuration } from './runDetailsUtils';
export { formatLabel } from './utils/formatting';
export { calculateBarPosition, calculateDuration, TIMELINE_CONSTANTS } from './utils/timing';
export { traceToTimelineData } from './utils/traceConversion';
