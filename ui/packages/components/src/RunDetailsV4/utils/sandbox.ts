/**
 * Sandbox-aware trace helpers. Steps behind `step.sandbox` carry
 * `inngest.sandbox` metadata (see docs/sandbox-traces/README.md); these
 * helpers read it to fold a statement's steps into one row and to describe
 * that row. Spans without the metadata are never touched.
 *
 * Grouping and linking rely on metadata only (`statement_id`, `sandbox_id`),
 * never on step IDs or names.
 */

import type { SandboxMetadata } from '../../generated';
import type { SandboxBarData, SandboxPhaseData } from '../TimelineBar.types';
import { isSandboxMetadata, type Trace } from '../types';

export function getSandboxMetadata(trace: Trace): SandboxMetadata | undefined {
  return trace.metadata?.find(isSandboxMetadata)?.values;
}

function sandboxEntries(traces: Trace[]): SandboxMetadata[] {
  return traces.flatMap((trace) => {
    const md = getSandboxMetadata(trace);
    return md ? [md] : [];
  });
}

function queuedAtMs(trace: Trace): number {
  return new Date(trace.queuedAt).getTime();
}

function latestEndedAt(traces: Trace[]): string | null {
  let latest: string | null = null;
  for (const trace of traces) {
    if (!trace.endedAt) {
      return null;
    }
    if (!latest || new Date(trace.endedAt) > new Date(latest)) {
      latest = trace.endedAt;
    }
  }
  return latest;
}

/**
 * The statement's result for the step panel: the latest member that produced
 * output (sleeps don't), with the statement's command and exit code filled in
 * from the other members, since the output step may not carry them itself.
 */
function statementMetadata(result: Trace, members: Trace[]): Trace['metadata'] {
  const entries = sandboxEntries(members);
  const command = entries.find((md) => md.command)?.command;
  const commandDisplay = entries.find((md) => md.command_display)?.command_display;
  const exitCode = entries.reverse().find((md) => md.exit_code !== undefined)?.exit_code;

  return result.metadata?.map((md) => {
    if (!isSandboxMetadata(md)) {
      return md;
    }
    return {
      ...md,
      values: {
        ...md.values,
        command: md.values.command ?? command,
        command_display: md.values.command_display ?? commandDisplay,
        exit_code: md.values.exit_code ?? exitCode,
      },
    };
  });
}

/**
 * Fold a statement's steps (two or more run children sharing a `statement_id`)
 * into one virtual span, the same way rollupStepAttempts folds attempts. The
 * span covers the earliest member's queue time to the latest member's end,
 * and is titled by the `role: "statement"` member when there is one. A CI
 * command has none: its steps are all internal, so the title falls back to
 * `statement_name`.
 *
 * The members aren't children: the row shows them as states of its bar, and
 * only the statement step's own attempts stay expandable.
 */
function rollupSandboxStatement(statementID: string, members: Trace[]): Trace {
  const sorted = [...members].sort((a, b) => queuedAtMs(a) - queuedAtMs(b));
  const first = sorted[0] as Trace;
  const last = sorted[sorted.length - 1] as Trace;
  const statement = sorted.find((m) => getSandboxMetadata(m)?.role === 'statement');
  const statementName = sorted
    .map((m) => getSandboxMetadata(m)?.statement_name)
    .find((name) => !!name);
  const result = [...sorted].reverse().find((m) => m.outputID) ?? last;
  const endedAt = latestEndedAt(sorted);

  return {
    isRoot: false,
    isUserland: false,
    spanID: `${statementID}-statement`, // virtual span
    groupID: statement?.groupID ?? last.groupID,
    name: statement?.name ?? statementName ?? first.name,
    attempts: statement?.attempts ?? null,
    stepID: statement?.stepID ?? first.stepID,
    stepOp: statement?.stepOp ?? 'RUN',
    stepType: statement?.stepType,
    queuedAt: first.queuedAt,
    scheduledAt: first.scheduledAt,
    startedAt: first.startedAt,
    endedAt,
    status: endedAt ? last.status : 'RUNNING',
    outputID: result.outputID,
    debugRunID: result.debugRunID,
    debugSessionID: result.debugSessionID,
    stepInfo: result.stepInfo,
    // Only the statement step's own attempts (or spans) stay expandable
    childrenSpans: statement?.childrenSpans ?? [],
    metadata: statementMetadata(result, sorted),
    userlandSpan: null,
    sandboxMembers: sorted,
  };
}

/**
 * Group the run's (already attempt-rolled-up) children by `statement_id`.
 * Groups with one member, and spans without sandbox metadata, pass through
 * unchanged.
 */
export function rollupSandboxStatements(children: Trace[]): Trace[] {
  const groups = new Map<string, Trace[]>();
  for (const child of children) {
    const statementID = getSandboxMetadata(child)?.statement_id;
    if (statementID) {
      groups.set(statementID, [...(groups.get(statementID) ?? []), child]);
    }
  }

  const result: Trace[] = [];
  for (const child of children) {
    const statementID = getSandboxMetadata(child)?.statement_id;
    const members = statementID ? groups.get(statementID) : undefined;
    if (!statementID || !members || members.length < 2) {
      result.push(child);
    } else if (members[0] === child) {
      result.push(rollupSandboxStatement(statementID, members));
    }
  }
  return result;
}

type Phase = Pick<SandboxPhaseData, 'key' | 'label' | 'waiting'>;

const RUNNING: Phase = { key: 'running', label: 'Running', waiting: true };

/**
 * The state a member step puts its statement in. Kept to a few states: every
 * poll and sleep of a running command is just "Running".
 */
function phaseOf(member: Trace, statement: string): Phase {
  const action = getSandboxMetadata(member)?.action ?? '';

  if (statement.startsWith('snapshot')) {
    return action === 'snapshot.create'
      ? { key: 'creating', label: 'Creating snapshot', waiting: false }
      : { key: 'waiting', label: 'Waiting until ready', waiting: true };
  }

  const isSleep = action === 'sleep' || member.stepOp?.toUpperCase() === 'SLEEP';
  if (isSleep) {
    return RUNNING;
  }
  if (action === 'create') {
    return { key: 'creating', label: 'Creating sandbox', waiting: false };
  }
  if (action.endsWith('.start')) {
    return { key: 'starting', label: 'Starting', waiting: false };
  }
  if (action.toLowerCase().includes('output')) {
    return { key: 'output', label: 'Collecting output', waiting: false };
  }
  return RUNNING;
}

/**
 * The states a statement moved through, from its member steps in time order.
 * Consecutive members in the same state merge, and each state runs until the
 * next one starts, so the states cover the whole row with no gaps.
 */
function statementPhases(
  members: Trace[],
  statement: string,
  endedAt: string | null
): SandboxPhaseData[] {
  const phases: SandboxPhaseData[] = [];
  for (const member of members) {
    const phase = phaseOf(member, statement);
    const previous = phases[phases.length - 1];
    if (previous?.key === phase.key) {
      continue;
    }

    const startTime = new Date(member.queuedAt);
    if (previous) {
      previous.endTime = startTime;
    }
    phases.push({ ...phase, startTime, endTime: null });
  }

  const final = phases[phases.length - 1];
  if (final) {
    final.endTime = endedAt ? new Date(endedAt) : null;
  }
  return phases;
}

/**
 * Describe a span for its row: machine, command, exit code, and for a
 * statement that owns several steps, its states. Undefined for spans without
 * sandbox metadata.
 */
export function getSandboxBarData(trace: Trace): SandboxBarData | undefined {
  const entries = sandboxEntries(trace.sandboxMembers ?? [trace]);
  if (entries.length === 0) {
    return undefined;
  }

  const statementEntry = entries.find((md) => md.role === 'statement') ?? entries[0]!;
  const sandboxId = entries.find((md) => md.sandbox_id)?.sandbox_id;
  const sandboxName = entries.find((md) => md.sandbox_name)?.sandbox_name;
  const withCommand = entries.find((md) => md.command_display || md.command?.length);
  const exitCode = [...entries].reverse().find((md) => md.exit_code !== undefined)?.exit_code;

  return {
    sandboxId,
    machineLabel: sandboxName ?? sandboxId?.slice(0, 8),
    statement: statementEntry.statement,
    command: withCommand?.command_display ?? withCommand?.command?.join(' '),
    exitCode,
    actions: entries.map((md) => md.action),
    phases: trace.sandboxMembers
      ? statementPhases(trace.sandboxMembers, statementEntry.statement, trace.endedAt)
      : undefined,
  };
}

/**
 * Mark the first row of each machine the run didn't create as "existing", and
 * annotate the rows. Rows must be the run's direct children in time order.
 */
export function annotateSandboxRows(rows: { sandbox?: SandboxBarData }[]): void {
  const created = new Set<string>();
  for (const row of rows) {
    if (row.sandbox?.sandboxId && row.sandbox.actions.includes('create')) {
      created.add(row.sandbox.sandboxId);
    }
  }

  const seen = new Set<string>();
  for (const row of rows) {
    if (!row.sandbox) {
      continue;
    }
    row.sandbox.annotate = true;

    const id = row.sandbox.sandboxId;
    if (id && !seen.has(id)) {
      seen.add(id);
      row.sandbox.existing = !created.has(id);
    }
  }
}

const STATEMENT_VERBS: Record<string, string> = {
  create: 'Create sandbox',
  get: 'Get sandbox',
  'commands.run': 'Run',
  'processes.start': 'Start',
  'process.wait': 'Wait for',
  'process.kill': 'Kill',
  snapshot: 'Snapshot',
  'snapshot.clone': 'Clone snapshot',
  destroy: 'Destroy',
};

/** Human verb for the SDK method, falling back to the method itself */
export function statementVerb(statement: string): string {
  return STATEMENT_VERBS[statement] ?? statement;
}
