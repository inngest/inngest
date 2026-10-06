/**
 * Sandbox machine chips and the machine highlight.
 *
 * A row backed by `inngest.sandbox` metadata shows a line under its name: the
 * command, a chip for its machine (coloured by when the machine first appears
 * in the run) and the exit code. A span group shows the chip when all its
 * sandbox steps ran on one machine, and its last exit code. Clicking a chip
 * pins a highlight on every row of that machine; hovering one previews it;
 * Escape clears it.
 *
 * Everything sandbox-specific in the timeline lives here.
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import { cn } from '../utils/classNames';
import type { TimelineBarData } from './TimelineBar.types';
import { isSandboxMetadata, isSpanGroup, type Trace } from './types';

export type SandboxBarData = {
  sandboxId?: string;
  machineLabel?: string;
  command?: string;
  exitCode?: number;
};

/** What a row shows about the sandbox work behind it, if any */
export function sandboxBarData(trace: Trace): SandboxBarData | undefined {
  if (!isSpanGroup(trace)) {
    const md = trace.metadata?.find(isSandboxMetadata)?.values;
    if (!md) return undefined;

    return {
      sandboxId: md.sandbox_id,
      machineLabel: md.sandbox_name ?? md.sandbox_id,
      command: md.command_display ?? md.command?.join(' '),
      exitCode: md.exit_code,
    };
  }

  // A group's sandbox steps, in order. Steps that name no machine (like a
  // snapshot readiness wait) don't count against a shared machine.
  const steps: SandboxBarData[] = [];
  const visit = (t: Trace) => {
    for (const child of t.childrenSpans ?? []) {
      if (isSpanGroup(child)) {
        visit(child);
      } else {
        const data = sandboxBarData(child);
        if (data) steps.push(data);
      }
    }
  };
  visit(trace);

  const machines = steps.filter((s) => s.sandboxId);
  const sharedMachine = machines.every((s) => s.sandboxId === machines[0]?.sandboxId)
    ? machines[0]
    : undefined;
  const exitCode = [...steps].reverse().find((s) => s.exitCode !== undefined)?.exitCode;
  if (!sharedMachine && exitCode === undefined) return undefined;

  return {
    sandboxId: sharedMachine?.sandboxId,
    machineLabel: sharedMachine?.machineLabel,
    exitCode,
  };
}

/** The run's machines in order of first appearance, each once */
export function collectMachineIds(bars: TimelineBarData[]): string[] {
  const ids = new Set<string>();
  const visit = (list: TimelineBarData[]) => {
    for (const bar of list) {
      if (bar.sandbox?.sandboxId) ids.add(bar.sandbox.sandboxId);
      visit(bar.children ?? []);
    }
  };
  visit(bars);
  return [...ids];
}

// Theme-aware chart colours, skipping green and red so a machine never reads
// as a status.
const MACHINE_COLORS = [
  '--color-chart-line-4',
  '--color-chart-line-5',
  '--color-chart-line-2',
  '--color-chart-line-3',
];

/** A machine's colour as an rgb() string, by its first appearance in the run */
export function machineColor(sandboxId: string, machineIds: string[], alpha = 1): string {
  const index = Math.max(0, machineIds.indexOf(sandboxId));
  return `rgb(var(${MACHINE_COLORS[index % MACHINE_COLORS.length]}) / ${alpha})`;
}

/**
 * Shorten a machine name in the middle, keeping its end: CI names machines
 * `ci-<runID>-<job>`, and the job is the part worth reading.
 */
export function shortMachineLabel(label: string, max = 20): string {
  if (label.length <= max) return label;
  const tail = max - 7;
  return `${label.slice(0, 6)}…${label.slice(-tail)}`;
}

// ============================================================================
// Highlight state
// ============================================================================

type MachineHighlightState = {
  machineIds: string[];
  /** The pinned machine, else the previewed one */
  activeId: string | null;
  pinnedId: string | null;
  togglePin: (sandboxId: string) => void;
  preview: (sandboxId: string | null) => void;
};

const MachineHighlightContext = createContext<MachineHighlightState>({
  machineIds: [],
  activeId: null,
  pinnedId: null,
  togglePin: () => {},
  preview: () => {},
});

export function MachineHighlightProvider({
  bars,
  children,
}: {
  bars: TimelineBarData[];
  children: ReactNode;
}) {
  const machineIds = useMemo(() => collectMachineIds(bars), [bars]);
  const [pinnedId, setPinnedId] = useState<string | null>(null);
  const [previewId, setPreviewId] = useState<string | null>(null);

  const togglePin = useCallback((sandboxId: string) => {
    setPinnedId((current) => (current === sandboxId ? null : sandboxId));
  }, []);

  useEffect(() => {
    if (!pinnedId) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setPinnedId(null);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [pinnedId]);

  const value = useMemo(
    () => ({
      machineIds,
      activeId: pinnedId ?? previewId,
      pinnedId,
      togglePin,
      preview: setPreviewId,
    }),
    [machineIds, pinnedId, previewId, togglePin]
  );

  return (
    <MachineHighlightContext.Provider value={value}>{children}</MachineHighlightContext.Provider>
  );
}

/** The machine of the nearest row above that has one, so sub-rows highlight too */
const RowMachineContext = createContext<string | undefined>(undefined);

export function useRowMachineId(ownId?: string): string | undefined {
  const inherited = useContext(RowMachineContext);
  return ownId ?? inherited;
}

export function MachineScope({ sandboxId, children }: { sandboxId?: string; children: ReactNode }) {
  return <RowMachineContext.Provider value={sandboxId}>{children}</RowMachineContext.Provider>;
}

/** Dotted background in the machine's colour while its rows are highlighted */
export function MachineHighlight({ sandboxId }: { sandboxId?: string }) {
  const { activeId, machineIds } = useContext(MachineHighlightContext);
  if (!sandboxId || sandboxId !== activeId) return null;

  return (
    <div
      data-testid="machine-highlight"
      className="pointer-events-none absolute inset-0"
      style={{
        backgroundImage: `radial-gradient(circle, ${machineColor(
          sandboxId,
          machineIds,
          0.6
        )} 1px, transparent 1px)`,
        backgroundSize: '7px 7px',
      }}
    />
  );
}

// ============================================================================
// Row annotation
// ============================================================================

function MachineChip({ sandboxId, label }: { sandboxId: string; label: string }) {
  const { machineIds, pinnedId, togglePin, preview } = useContext(MachineHighlightContext);
  const pinned = pinnedId === sandboxId;
  const color = (alpha?: number) => machineColor(sandboxId, machineIds, alpha);

  return (
    <button
      type="button"
      data-testid="machine-chip"
      aria-pressed={pinned}
      title={`${label}\n${
        pinned ? 'Clear machine highlight' : 'Highlight every row on this machine'
      }`}
      className="text-subtle inline-flex h-4 min-w-0 shrink items-center gap-1 rounded-full border px-1.5 font-mono text-[11px] leading-none"
      style={{
        borderColor: color(pinned ? 1 : 0.6),
        backgroundColor: color(pinned ? 0.25 : 0.1),
      }}
      // The chip sits inside a clickable row; don't select or toggle the row
      onClick={(e) => {
        e.stopPropagation();
        togglePin(sandboxId);
      }}
      onMouseDown={(e) => e.stopPropagation()}
      onMouseEnter={() => preview(sandboxId)}
      onMouseLeave={() => preview(null)}
    >
      <span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ backgroundColor: color() }} />
      <span className="truncate">{shortMachineLabel(label)}</span>
    </button>
  );
}

function ExitBadge({ exitCode }: { exitCode: number }) {
  return (
    <span
      data-testid="exit-badge"
      className={cn(
        'shrink-0 rounded px-1 font-mono text-[11px] leading-4',
        exitCode === 0
          ? 'bg-primary-3xSubtle text-primary-intense'
          : 'bg-tertiary-3xSubtle text-tertiary-intense'
      )}
    >
      exit {exitCode}
    </span>
  );
}

/** The line under a sandbox row's name */
export function SandboxAnnotation({
  sandbox,
  className,
}: {
  sandbox: SandboxBarData;
  className?: string;
}) {
  const { sandboxId, machineLabel, command, exitCode } = sandbox;

  return (
    <span
      data-testid="sandbox-annotation"
      className={cn(
        'text-light flex min-w-0 items-center gap-1 overflow-hidden whitespace-nowrap text-xs',
        className
      )}
    >
      {command && (
        <code className="text-subtle min-w-0 truncate font-mono text-[11px]" title={command}>
          {command}
        </code>
      )}
      {command && sandboxId && <span className="shrink-0">on</span>}
      {sandboxId && <MachineChip sandboxId={sandboxId} label={machineLabel ?? sandboxId} />}
      {exitCode !== undefined && <ExitBadge exitCode={exitCode} />}
    </span>
  );
}
