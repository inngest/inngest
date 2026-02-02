/**
 * RunDetailsV4 - Composable Timeline Bar Component
 * Feature: 001-composable-timeline-bar
 *
 * This module provides a single, ultra-composable TimelineBar component
 * for rendering timeline visualizations in function run details.
 */

// Types (T004)
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

// Components (T013, T038)
export { TimelineBar } from './TimelineBar';
export { TimelineHeader } from './TimelineHeader';
export { Timeline } from './Timeline';
export { RunDetailsV4 } from './RunDetailsV4';

// Utilities (T006, T007)
export { formatDuration, formatLabel } from './utils/formatting';
export { calculateBarPosition, calculateDuration, TIMELINE_CONSTANTS } from './utils/timing';
export { traceToTimelineData } from './utils/traceConversion';
