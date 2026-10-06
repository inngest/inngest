/**
 * @module
 * The muted `S1`, `S2`, … badge before a row's name, saying which sandbox
 * (from `inngest.sandbox` metadata) the row's work ran on. Everything
 * sandbox-specific in the timeline lives here.
 */

import { createContext, useContext, useMemo, type ReactNode } from 'react';

import type { TimelineBarData } from './TimelineBar.types';
import { isSandboxMetadata, isSpanGroup, type Trace } from './types';

export type SandboxBarData = {
  sandboxId: string;
  /** `sandbox_name`, else `sandbox_id`, for the badge's tooltip */
  label: string;
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

    return { sandboxId: md.sandbox_id, label: md.sandbox_name ?? md.sandbox_id };
  }

  const votes = new Map<string, { sandbox: SandboxBarData; count: number }>();

  for (const child of trace.childrenSpans ?? []) {
    const sandbox = sandboxBarData(child);
    if (!sandbox) continue;

    const count = (votes.get(sandbox.sandboxId)?.count ?? 0) + 1;
    votes.set(sandbox.sandboxId, { sandbox, count });
  }

  const [first, second] = [...votes.values()].sort((a, b) => {
    return b.count - a.count;
  });

  if (!first || second?.count === first.count) return undefined;

  return first.sandbox;
}

/**
 * The run's sandboxes numbered from 1 in reading order: siblings by start
 * time, and a row's own sandbox before the ones its children add. A job is
 * then numbered before an extra sandbox it starts, even one that starts first.
 */
export function numberSandboxes(bars: TimelineBarData[]): Map<string, number> {
  const numbers = new Map<string, number>();

  const visit = (list: TimelineBarData[]) => {
    const byStart = [...list].sort((a, b) => {
      return a.startTime.getTime() - b.startTime.getTime();
    });

    for (const bar of byStart) {
      const id = bar.sandbox?.sandboxId;

      if (id && !numbers.has(id)) {
        numbers.set(id, numbers.size + 1);
      }

      visit(bar.children ?? []);
    }
  };

  visit(bars);

  return numbers;
}

const SandboxNumbers = createContext<Map<string, number>>(new Map());

export function SandboxNumbersProvider({
  bars,
  children,
}: {
  bars: TimelineBarData[];
  children: ReactNode;
}) {
  const numbers = useMemo(() => {
    return numberSandboxes(bars);
  }, [bars]);

  return <SandboxNumbers.Provider value={numbers}>{children}</SandboxNumbers.Provider>;
}

/** The sandbox of the nearest row above that has one */
const ParentSandbox = createContext<string | undefined>(undefined);

/** Wraps a row's children so they know the sandbox their row is on */
export function SandboxScope({
  sandbox,
  children,
}: {
  sandbox?: SandboxBarData;
  children: ReactNode;
}) {
  const parent = useContext(ParentSandbox);

  return (
    <ParentSandbox.Provider value={sandbox?.sandboxId ?? parent}>{children}</ParentSandbox.Provider>
  );
}

/** `S1` before a row's name, unless the row above is on the same sandbox */
export function SandboxBadge({ sandbox }: { sandbox?: SandboxBarData }) {
  const parent = useContext(ParentSandbox);
  const numbers = useContext(SandboxNumbers);

  const number = sandbox && numbers.get(sandbox.sandboxId);
  if (!sandbox || !number || sandbox.sandboxId === parent) return null;

  return (
    <span
      data-testid="sandbox-badge"
      title={sandbox.label}
      className="text-light mr-1.5 align-middle font-mono text-[11px]"
    >
      S{number}
    </span>
  );
}
