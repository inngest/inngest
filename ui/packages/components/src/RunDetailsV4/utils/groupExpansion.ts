/**
 * Which span groups start expanded. A group is a container for the real steps
 * inside it, so it starts collapsed and reads as one row; the path to a failed
 * or running descendant starts open so a problem is visible without a click.
 * @module
 */

import type { TimelineBarData } from '../TimelineBar.types';

const attentionStatuses = new Set(['FAILED', 'RUNNING']);

/**
 * Collect the IDs of the span groups that contain a failed or running
 * descendant at any depth. Rows that aren't span groups never appear.
 */
export function collectAutoExpandedGroupIds(bars: TimelineBarData[]): Set<string> {
  const ids = new Set<string>();

  // Whether `bar` or anything under it needs attention
  const visit = (bar: TimelineBarData): boolean => {
    let descendantNeedsAttention = false;

    for (const child of bar.children ?? []) {
      if (visit(child)) {
        descendantNeedsAttention = true;
      }
    }

    if (descendantNeedsAttention && bar.style === 'span.group') {
      ids.add(bar.id);
    }

    return descendantNeedsAttention || attentionStatuses.has(bar.status ?? '');
  };

  for (const bar of bars) {
    visit(bar);
  }

  return ids;
}
