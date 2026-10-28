/**
 * @module
 * Which sandbox (from `inngest.sandbox` metadata) a row's work ran on, and the
 * muted `S1`, `S2`, … badge that says so.
 */

import type { TimelineBarData } from '../TimelineBar.types';
import { isSandboxMetadata, isSpanGroup, type Trace } from '../types';

export type SandboxBarData = {
  id: string;
  /** `sandbox_name`, else `sandbox_id`, for the badge's tooltip */
  label: string;
  /** `S1`, `S2`, …; absent when the row above is on the same sandbox */
  badge?: string;
};

/**
 * A step's own sandbox, or the one most of a group's direct children use.
 * Each child votes once (a subgroup with its own majority), children without
 * a sandbox don't vote, and a tie means no sandbox.
 */
export function sandboxBarData(trace: Trace): SandboxBarData | undefined {
  if (!isSpanGroup(trace)) {
    const md = trace.metadata?.find(isSandboxMetadata)?.values;
    if (!md?.sandbox_id) return undefined;

    return { id: md.sandbox_id, label: md.sandbox_name ?? md.sandbox_id };
  }

  const votes = new Map<string, { sandbox: SandboxBarData; count: number }>();

  for (const child of trace.childrenSpans ?? []) {
    const sandbox = sandboxBarData(child);
    if (!sandbox) continue;

    const count = (votes.get(sandbox.id)?.count ?? 0) + 1;
    votes.set(sandbox.id, { sandbox, count });
  }

  const [first, second] = [...votes.values()].sort((a, b) => {
    return b.count - a.count;
  });

  if (!first || second?.count === first.count) return undefined;

  return first.sandbox;
}

/**
 * Numbers the run's sandboxes from 1 in reading order (siblings by start time,
 * a row's own sandbox before the ones its children add) and gives each row a
 * badge unless its parent is on the same sandbox. A job is then numbered
 * before an extra sandbox it starts, even one that starts first.
 */
export function badgeSandboxes(bars: TimelineBarData[]): void {
  const numbers = new Map<string, number>();

  const visit = (list: TimelineBarData[], parentID?: string) => {
    const byStart = [...list].sort((a, b) => {
      return a.startTime.getTime() - b.startTime.getTime();
    });

    for (const bar of byStart) {
      const sandbox = bar.sandbox;

      if (sandbox) {
        if (!numbers.has(sandbox.id)) {
          numbers.set(sandbox.id, numbers.size + 1);
        }

        if (sandbox.id !== parentID) {
          bar.sandbox = { ...sandbox, badge: `S${numbers.get(sandbox.id)}` };
        }
      }

      visit(bar.children ?? [], sandbox?.id ?? parentID);
    }
  };

  visit(bars);
}
