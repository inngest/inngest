/**
 * Formatting utilities for the TimelineBar component.
 * Feature: 001-composable-timeline-bar
 */

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
