/**
 * Timing calculation utilities for the TimelineBar component.
 * Feature: 001-composable-timeline-bar
 */

/**
 * Layout constants for the TimelineBar component.
 */
export const TIMELINE_CONSTANTS = {
  /** Minimum bar width in pixels for visibility */
  MIN_BAR_WIDTH_PX: 2,

  /** Indentation per depth level in pixels */
  INDENT_WIDTH_PX: 20,

  /** Base left padding for the left panel in pixels */
  BASE_LEFT_PADDING_PX: 4,

  /** Row height in pixels */
  ROW_HEIGHT_PX: 28,

  /** Transition duration for expand/collapse */
  TRANSITION_MS: 150,

  /** Default left panel width percentage */
  DEFAULT_LEFT_WIDTH: 40,
} as const;

/**
 * Calculate bar position as percentages of timeline width.
 * @param startTime - Start time of the bar
 * @param endTime - End time of the bar (null if in progress)
 * @param minTime - Minimum time of the overall timeline
 * @param maxTime - Maximum time of the overall timeline
 * @returns Object with startPercent and widthPercent
 */
export function calculateBarPosition(
  startTime: Date,
  endTime: Date | null,
  minTime: Date,
  maxTime: Date
): { startPercent: number; widthPercent: number } {
  const totalMs = maxTime.getTime() - minTime.getTime();

  if (totalMs <= 0) {
    return { startPercent: 0, widthPercent: 100 };
  }

  const startMs = startTime.getTime() - minTime.getTime();
  const endMs = (endTime ?? new Date()).getTime() - minTime.getTime();

  const startPercent = (startMs / totalMs) * 100;
  const widthPercent = Math.max(0.1, ((endMs - startMs) / totalMs) * 100); // Min 0.1%

  return { startPercent, widthPercent };
}

/**
 * Calculate the duration in milliseconds between two dates.
 * @param startTime - Start time
 * @param endTime - End time (null uses current time)
 * @returns Duration in milliseconds
 */
export function calculateDuration(startTime: Date, endTime: Date | null): number {
  const end = endTime ?? new Date();
  return Math.max(0, end.getTime() - startTime.getTime());
}
