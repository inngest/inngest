/**
 * Sandbox row annotations and the machine highlight.
 *
 * A row backed by `inngest.sandbox` metadata shows a line under its name: what
 * the statement did, a chip for its machine (coloured by when the machine
 * first appears in the run), and the exit code. Clicking a chip pins a highlight on every row of that
 * machine; hovering one previews it while nothing is pinned.
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type CSSProperties,
  type ReactNode,
} from 'react';

import { cn } from '../utils/classNames';
import type { SandboxBarData } from './TimelineBar.types';
import { statementVerb } from './utils/sandbox';

// Theme-aware chart colours, skipping green and red so a machine never reads
// as a status. Ordered so neighbours differ most: purple, orange, blue, yellow.
const MACHINE_COLORS = [
  '--color-chart-line-4',
  '--color-chart-line-2',
  '--color-chart-line-5',
  '--color-chart-line-3',
];

/**
 * Colour for a machine, as an rgb() string with the given alpha. Machines are
 * coloured in order of first appearance in the run (`machineIds`), so
 * neighbouring machines never share or nearly share a colour. A machine not in
 * the list falls back to a hash of its ID.
 */
export function machineColor(sandboxId: string, machineIds: string[] = [], alpha = 1): string {
  let index = machineIds.indexOf(sandboxId);
  if (index < 0) {
    // FNV-1a
    let hash = 0x811c9dc5;
    for (let i = 0; i < sandboxId.length; i++) {
      hash ^= sandboxId.charCodeAt(i);
      hash = Math.imul(hash, 0x01000193);
    }
    index = hash >>> 0;
  }
  const token = MACHINE_COLORS[index % MACHINE_COLORS.length];
  return `rgb(var(${token}) / ${alpha})`;
}

/**
 * Shorten a machine name in the middle, keeping its end: CI names machines
 * `ci-<runID>-<job>`, and the job is the part worth reading
 * (`ci-01M…-base`).
 */
export function shortMachineLabel(label: string, max = 20): string {
  if (label.length <= max) {
    return label;
  }
  const head = label.slice(0, 6);
  const budget = max - head.length - 1;
  // The longest dash-delimited ending that fits, else the last few characters
  let tail = label.slice(-budget);
  for (let i = label.indexOf('-', head.length); i >= 0; i = label.indexOf('-', i + 1)) {
    if (label.length - i <= budget) {
      tail = label.slice(i);
      break;
    }
  }
  return `${head}…${tail}`;
}

// ============================================================================
// Highlight state
// ============================================================================

type SandboxHighlight = {
  /** The run's machines in order of first appearance, for colours */
  machineIds: string[];
  /** Machine whose rows are highlighted: the pinned one, else the previewed one */
  activeId: string | null;
  pinnedId: string | null;
  togglePin: (sandboxId: string) => void;
  preview: (sandboxId: string | null) => void;
};

const SandboxHighlightContext = createContext<SandboxHighlight>({
  machineIds: [],
  activeId: null,
  pinnedId: null,
  togglePin: () => {},
  preview: () => {},
});

export function useSandboxHighlight(): SandboxHighlight {
  return useContext(SandboxHighlightContext);
}

export function SandboxHighlightProvider({
  children,
  machineIds = [],
}: {
  children: ReactNode;
  /** The run's machines in order of first appearance */
  machineIds?: string[];
}) {
  const [pinnedId, setPinnedId] = useState<string | null>(null);
  const [previewId, setPreviewId] = useState<string | null>(null);

  const togglePin = useCallback((sandboxId: string) => {
    setPinnedId((current) => (current === sandboxId ? null : sandboxId));
  }, []);

  useEffect(() => {
    if (!pinnedId) {
      return;
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setPinnedId(null);
      }
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
    <SandboxHighlightContext.Provider value={value}>{children}</SandboxHighlightContext.Provider>
  );
}

// ============================================================================
// Row pieces
// ============================================================================

/**
 * Dotted row background. Experiments use the neutral default; a highlighted
 * machine uses its own colour.
 */
export function DottedBackground({ color }: { color?: string }) {
  const style: CSSProperties = {
    backgroundImage: `radial-gradient(circle, ${
      color ?? 'rgb(var(--color-border-muted))'
    } 1px, transparent 1px)`,
    backgroundSize: '7px 7px',
  };
  return (
    <div
      data-testid="dotted-background"
      className={cn('pointer-events-none absolute inset-0', !color && 'bg-canvasSubtle')}
      style={style}
    />
  );
}

function MachineChip({
  sandboxId,
  label,
  existing,
}: {
  sandboxId: string;
  label: string;
  existing?: boolean;
}) {
  const { machineIds, pinnedId, togglePin, preview } = useSandboxHighlight();
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
        boxShadow: pinned ? `0 0 0 1px ${color()}` : undefined,
      }}
      // The chip lives inside a clickable row; don't select or toggle it too
      onClick={(e) => {
        e.stopPropagation();
        togglePin(sandboxId);
      }}
      onMouseDown={(e) => e.stopPropagation()}
      onMouseEnter={() => preview(sandboxId)}
      onMouseLeave={() => preview(null)}
    >
      <span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ backgroundColor: color() }} />
      <span className="truncate">
        {shortMachineLabel(label)}
        {existing && <span className="text-light"> · existing</span>}
      </span>
    </button>
  );
}

function ExitBadge({ exitCode, attempts }: { exitCode?: number; attempts?: number }) {
  const parts = [
    exitCode !== undefined && `exit ${exitCode}`,
    attempts && attempts > 1 && `${attempts} attempts`,
  ].filter(Boolean);
  const passed = exitCode === undefined || exitCode === 0;
  return (
    <span
      data-testid="exit-badge"
      className={cn(
        'shrink-0 rounded px-1 font-mono text-[11px] leading-4',
        passed
          ? 'bg-primary-3xSubtle text-primary-intense'
          : 'bg-tertiary-3xSubtle text-tertiary-intense'
      )}
    >
      {parts.join(' · ')}
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
  const { sandboxId, machineLabel, exitCode, attempts, existing } = sandbox;
  // CI titles a command's row with the command, so don't repeat it
  const command = sandbox.commandInTitle ? undefined : sandbox.command;

  return (
    <span
      data-testid="sandbox-annotation"
      className={cn(
        'text-light flex min-w-0 items-center gap-1 overflow-hidden whitespace-nowrap text-xs',
        className
      )}
    >
      <span className="shrink-0">{statementVerb(sandbox.statement)}</span>
      {command && (
        <code className="text-subtle min-w-0 truncate font-mono text-[11px]">{command}</code>
      )}
      {sandbox.command && sandboxId && <span className="shrink-0">on</span>}
      {sandboxId && (
        <MachineChip sandboxId={sandboxId} label={machineLabel ?? sandboxId} existing={existing} />
      )}
      {(exitCode !== undefined || (attempts ?? 0) > 1) && (
        <ExitBadge exitCode={exitCode} attempts={attempts} />
      )}
    </span>
  );
}
