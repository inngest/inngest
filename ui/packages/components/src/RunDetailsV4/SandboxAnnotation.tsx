/**
 * Sandbox machine tags and the machine highlight, from `inngest.sandbox`
 * metadata. Everything sandbox-specific in the timeline lives here.
 */

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from 'react';

import { cn } from '../utils/classNames';
import { SpanKindTag } from './SpanKindTag';
import type { TimelineBarData } from './TimelineBar.types';
import { isSandboxMetadata, isSpanGroup, type Trace } from './types';

export type SandboxBarData = {
  sandboxId?: string;
  machineLabel?: string;
  command?: string;
};

/**
 * What a row shows about its sandbox work: a step's command and machine, or a
 * group's shared machine.
 */
export function sandboxBarData(trace: Trace): SandboxBarData | undefined {
  if (!isSpanGroup(trace)) {
    const md = trace.metadata?.find(isSandboxMetadata)?.values;
    if (!md) return undefined;

    return {
      sandboxId: md.sandbox_id,
      machineLabel: md.sandbox_name ?? md.sandbox_id,
      command: md.command_display ?? md.command?.join(' '),
    };
  }

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

  // Steps that name no machine (like a snapshot readiness wait) don't count
  const machine = steps.find((s) => s.sandboxId);
  const shared = steps.every((s) => !s.sandboxId || s.sandboxId === machine?.sandboxId)
    ? machine
    : undefined;
  if (!shared) return undefined;

  return { sandboxId: shared.sandboxId, machineLabel: shared.machineLabel };
}

// Theme-aware chart colours, skipping green and red so a machine never reads
// as a status.
const MACHINE_COLORS = [
  '--color-chart-line-4',
  '--color-chart-line-5',
  '--color-chart-line-2',
  '--color-chart-line-3',
];

/** A machine's colour, by its first appearance in the run */
function machineColor(sandboxId: string, machineIds: string[], alpha = 1): string {
  const index = Math.max(0, machineIds.indexOf(sandboxId));
  return `rgb(var(${MACHINE_COLORS[index % MACHINE_COLORS.length]}) / ${alpha})`;
}

/**
 * Shorten a machine name in the middle, keeping its end: CI names machines
 * `ci-<runID>-<job>`, and the job is the part worth reading.
 */
export function shortMachineLabel(label: string, max = 20): string {
  if (label.length <= max) return label;
  return `${label.slice(0, 6)}…${label.slice(-(max - 7))}`;
}

type MachineHighlightState = {
  machineIds: string[];
  pinnedId: string | null;
  previewId: string | null;
  setPinnedId: Dispatch<SetStateAction<string | null>>;
  setPreviewId: Dispatch<SetStateAction<string | null>>;
};

const MachineHighlightContext = createContext<MachineHighlightState>({
  machineIds: [],
  pinnedId: null,
  previewId: null,
  setPinnedId: () => {},
  setPreviewId: () => {},
});

export function MachineHighlightProvider({
  bars,
  children,
}: {
  bars: TimelineBarData[];
  children: ReactNode;
}) {
  // The run's machines in order of first appearance
  const machineIds = useMemo(() => {
    const ids = new Set<string>();
    const visit = (list: TimelineBarData[]) => {
      for (const bar of list) {
        if (bar.sandbox?.sandboxId) ids.add(bar.sandbox.sandboxId);
        visit(bar.children ?? []);
      }
    };
    visit(bars);
    return [...ids];
  }, [bars]);
  const [pinnedId, setPinnedId] = useState<string | null>(null);
  const [previewId, setPreviewId] = useState<string | null>(null);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setPinnedId(null);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  const value = useMemo(
    () => ({ machineIds, pinnedId, previewId, setPinnedId, setPreviewId }),
    [machineIds, pinnedId, previewId]
  );

  return (
    <MachineHighlightContext.Provider value={value}>{children}</MachineHighlightContext.Provider>
  );
}

/** The machine of the nearest row above that has one, so sub-rows highlight too */
const RowMachineContext = createContext<string | undefined>(undefined);

export const MachineScope = RowMachineContext.Provider;

export function useRowMachineId(ownId?: string): string | undefined {
  const inherited = useContext(RowMachineContext);
  return ownId ?? inherited;
}

/** Dotted background in the machine's colour while its rows are highlighted */
export function MachineHighlight({ sandboxId }: { sandboxId?: string }) {
  const { machineIds, pinnedId, previewId } = useContext(MachineHighlightContext);
  if (!sandboxId || sandboxId !== (pinnedId ?? previewId)) return null;

  const dot = machineColor(sandboxId, machineIds, 0.6);
  return (
    <div
      data-testid="machine-highlight"
      className="pointer-events-none absolute inset-0"
      style={{
        backgroundImage: `radial-gradient(circle, ${dot} 1px, transparent 1px)`,
        backgroundSize: '7px 7px',
      }}
    />
  );
}

/** A MACHINE tag and the machine's name; clicking pins its highlight */
function MachineTag({ sandboxId, label }: { sandboxId: string; label: string }) {
  const { pinnedId, setPinnedId, setPreviewId } = useContext(MachineHighlightContext);
  const pinned = pinnedId === sandboxId;

  return (
    <button
      type="button"
      data-testid="machine-tag"
      aria-pressed={pinned}
      title={`${label}\n${
        pinned ? 'Clear machine highlight' : 'Highlight every row on this machine'
      }`}
      className={cn(
        'text-subtle hover:text-basis inline-flex min-w-0 shrink items-center gap-1.5 font-mono text-[11px] leading-none',
        pinned && 'text-basis'
      )}
      // The tag sits inside a clickable row; don't select or toggle the row
      onClick={(e) => {
        e.stopPropagation();
        setPinnedId((current) => (current === sandboxId ? null : sandboxId));
      }}
      onMouseDown={(e) => e.stopPropagation()}
      onMouseEnter={() => setPreviewId(sandboxId)}
      onMouseLeave={() => setPreviewId(null)}
    >
      <SpanKindTag kind="machine" />
      <span className="truncate">{shortMachineLabel(label)}</span>
    </button>
  );
}

/** The line under a sandbox row's name */
export function SandboxAnnotation({
  sandbox: { sandboxId, machineLabel, command },
  className,
}: {
  sandbox: SandboxBarData;
  className?: string;
}) {
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
      {sandboxId && <MachineTag sandboxId={sandboxId} label={machineLabel ?? sandboxId} />}
    </span>
  );
}
