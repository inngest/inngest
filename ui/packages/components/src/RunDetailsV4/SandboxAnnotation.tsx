/**
 * Sandbox row annotations and the machine highlight.
 *
 * A row backed by `inngest.sandbox` metadata shows a line under its name: what
 * the statement did, a chip for its machine (coloured from `sandbox_id`), and
 * the exit code. Clicking a chip pins a highlight on every row of that
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
// as a status.
const MACHINE_COLORS = [
  '--color-chart-line-4',
  '--color-chart-line-2',
  '--color-chart-line-3',
  '--color-chart-line-5',
];

/** Deterministic colour for a machine, as an rgb() string with the given alpha */
export function machineColor(sandboxId: string, alpha = 1): string {
  // FNV-1a, so the same machine gets the same colour in every run
  let hash = 0x811c9dc5;
  for (let i = 0; i < sandboxId.length; i++) {
    hash ^= sandboxId.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  const token = MACHINE_COLORS[(hash >>> 0) % MACHINE_COLORS.length];
  return `rgb(var(${token}) / ${alpha})`;
}

// ============================================================================
// Highlight state
// ============================================================================

type SandboxHighlight = {
  /** Machine whose rows are highlighted: the pinned one, else the previewed one */
  activeId: string | null;
  pinnedId: string | null;
  togglePin: (sandboxId: string) => void;
  preview: (sandboxId: string | null) => void;
};

const SandboxHighlightContext = createContext<SandboxHighlight>({
  activeId: null,
  pinnedId: null,
  togglePin: () => {},
  preview: () => {},
});

export function useSandboxHighlight(): SandboxHighlight {
  return useContext(SandboxHighlightContext);
}

export function SandboxHighlightProvider({ children }: { children: ReactNode }) {
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
    () => ({ activeId: pinnedId ?? previewId, pinnedId, togglePin, preview: setPreviewId }),
    [pinnedId, previewId, togglePin]
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
  const { pinnedId, togglePin, preview } = useSandboxHighlight();
  const pinned = pinnedId === sandboxId;

  return (
    <button
      type="button"
      data-testid="machine-chip"
      aria-pressed={pinned}
      title={pinned ? 'Clear machine highlight' : 'Highlight every row on this machine'}
      className="text-subtle inline-flex h-4 min-w-0 shrink items-center gap-1 rounded-full border px-1.5 font-mono text-[11px] leading-none"
      style={{
        borderColor: machineColor(sandboxId, pinned ? 1 : 0.6),
        backgroundColor: machineColor(sandboxId, pinned ? 0.25 : 0.1),
        boxShadow: pinned ? `0 0 0 1px ${machineColor(sandboxId)}` : undefined,
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
      <span
        className="h-1.5 w-1.5 shrink-0 rounded-full"
        style={{ backgroundColor: machineColor(sandboxId) }}
      />
      <span className="truncate">
        {label}
        {existing && <span className="text-light"> · existing</span>}
      </span>
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
  const { sandboxId, machineLabel, command, exitCode, existing } = sandbox;

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
        <>
          <code className="text-subtle min-w-0 truncate font-mono text-[11px]">{command}</code>
          {sandboxId && <span className="shrink-0">on</span>}
        </>
      )}
      {sandboxId && (
        <MachineChip sandboxId={sandboxId} label={machineLabel ?? sandboxId} existing={existing} />
      )}
      {exitCode !== undefined && <ExitBadge exitCode={exitCode} />}
    </span>
  );
}
