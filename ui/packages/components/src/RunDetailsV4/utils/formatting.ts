/**
 * Formatting utilities for the TimelineBar component.
 * Feature: 001-composable-timeline-bar
 */

/**
 * Format duration in human-readable form.
 * @param ms - Duration in milliseconds
 * @returns Formatted string (e.g., "1.23s", "45.67ms", "<1ms")
 */
export function formatDuration(ms: number): string {
  if (ms < 1) return '<1ms';
  if (ms < 1000) return `${ms.toFixed(2)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  return `${(ms / 60000).toFixed(2)}m`;
}

/**
 * Format a label according to the specified format.
 * @param label - The label to format
 * @param format - The format to apply ('uppercase', 'titlecase', 'default')
 * @returns Formatted label
 */
export function formatLabel(label: string, format?: 'uppercase' | 'titlecase' | 'default'): string {
  switch (format) {
    case 'uppercase':
      return label.toUpperCase();
    case 'titlecase':
      return label.charAt(0).toUpperCase() + label.slice(1).toLowerCase();
    default:
      return label;
  }
}
