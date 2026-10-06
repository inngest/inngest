/**
 * A hand-built run trace shaped like the API's output for a CI-like run that
 * uses span groups, for tests and for previewing the timeline:
 *
 * - a `machine` group (create + setup) on machine A
 * - a group per command, one retried in `Attempt 1`/`Attempt 2` subgroups,
 *   with the retry check between attempts outside the command's group
 * - a background `dev server` group, re-entered to kill it at the end
 * - a snapshot whose readiness wait names no machine
 * - an ordinary step outside any group
 * - an agent → tool → steps nest three groups deep, with a retried step
 */

import type { SandboxMetadata } from '../../generated';
import type { Trace } from '../types';

const RUN_START = Date.parse('2026-10-01T12:00:00Z');
const at = (secs: number) => new Date(RUN_START + secs * 1000).toISOString();

export const MACHINE_A = { sandbox_id: 'sb_0a1f', sandbox_name: 'ci-01JB7Q2XKZ-base' };
const MACHINE_B = { sandbox_id: 'sb_7c3e', sandbox_name: 'ci-01JB7Q2XKZ-e2e' };

type StepOptions = {
  status?: string;
  stepOp?: string;
  attempts?: number;
  sandbox?: Omit<SandboxMetadata, 'version'>;
};

function step(
  id: string,
  name: string,
  from: number,
  to: number | null,
  { status = 'COMPLETED', stepOp = 'RUN', attempts = 0, sandbox }: StepOptions = {}
): Trace {
  return {
    attempts,
    endedAt: to === null ? null : at(to),
    isRoot: false,
    isUserland: false,
    name,
    outputID: to === null ? null : `out-${id}-${attempts}`,
    queuedAt: at(from),
    scheduledAt: at(from),
    startedAt: at(from),
    spanID: `span-${id}-${attempts}`,
    status,
    stepID: id,
    stepInfo: null,
    stepOp,
    stepType: stepOp,
    userlandSpan: null,
    metadata: sandbox
      ? [
          {
            kind: 'inngest.sandbox',
            scope: 'step',
            updatedAt: at(to ?? from),
            values: { version: 1, ...sandbox },
          },
        ]
      : [],
  };
}

const exec = (
  command: string,
  exitCode: number,
  machine = MACHINE_A
): Omit<SandboxMetadata, 'version'> => ({
  action: 'exec',
  method: 'commands.run',
  command_display: command,
  exit_code: exitCode,
  ...machine,
});

/** A group as the loader builds it: children by queue time, times and status from them */
function group(id: string, name: string, children: Trace[]): Trace {
  const sorted = [...children].sort((a, b) => Date.parse(a.queuedAt) - Date.parse(b.queuedAt));
  const last = sorted.reduce((a, b) =>
    Date.parse(b.endedAt ?? '') > Date.parse(a.endedAt ?? '') ? b : a
  );

  return {
    attempts: null,
    childrenSpans: sorted,
    endedAt: last.endedAt,
    isRoot: false,
    isUserland: false,
    name,
    outputID: null,
    queuedAt: sorted[0]!.queuedAt,
    scheduledAt: null,
    startedAt: sorted[0]!.startedAt,
    spanID: `span:${id}`,
    status: last.status,
    stepID: null,
    stepInfo: null,
    stepOp: null,
    stepType: 'SPAN_GROUP',
    userlandSpan: null,
    metadata: [],
  };
}

export const stepSpansTrace: Trace = {
  attempts: 0,
  childrenSpans: [
    group('machine', 'machine', [
      step('create', 'create', 0, 6, {
        sandbox: { action: 'create', method: 'create', ...MACHINE_A },
      }),
      step('setup', 'setup', 6, 14, { sandbox: exec('pnpm install --frozen-lockfile', 0) }),
    ]),
    group('lint', 'lint', [step('lint', 'pnpm lint', 14, 22, { sandbox: exec('pnpm lint', 0) })]),
    group('dev-server', 'dev server', [
      step('dev-start', 'start', 22, 24, {
        sandbox: { ...exec('pnpm dev', 0), action: 'process.start', process_id: 'p_1' },
      }),
      step('dev-ready', 'wait for port', 24, 30, { stepOp: 'SLEEP' }),
      step('dev-kill', 'kill', 104, 106, {
        sandbox: { ...exec('pnpm dev', 143), action: 'process.kill', process_id: 'p_1' },
      }),
    ]),
    group('test', 'test', [
      group('test-1', 'Attempt 1', [
        step('test-a1', 'pnpm test', 30, 52, {
          status: 'FAILED',
          sandbox: exec('pnpm test', 1),
        }),
      ]),
      group('test-2', 'Attempt 2', [
        step('test-a2', 'pnpm test', 58, 78, { sandbox: exec('pnpm test', 0) }),
      ]),
    ]),
    step('retry-check', 'check retry', 52, 58),
    group('snapshot', 'snapshot', [
      step('snap-create', 'create snapshot', 78, 82, {
        sandbox: { action: 'snapshot.create', method: 'snapshot', ...MACHINE_A },
      }),
      step('snap-ready', 'wait until ready', 82, 90, {
        stepOp: 'SLEEP',
        sandbox: { action: 'snapshot.waitUntilReady', method: 'snapshot', snapshot_id: 'snap_1' },
      }),
    ]),
    group('e2e', 'e2e', [
      step('e2e-create', 'create', 90, 94, {
        sandbox: { action: 'create', method: 'create', ...MACHINE_B },
      }),
      step('e2e', 'pnpm e2e', 94, 104, { sandbox: exec('pnpm e2e', 0, MACHINE_B) }),
    ]),
    step('notify', 'notify', 106, 108),
    group('network', 'Research network', [
      group('agent', 'Research agent', [
        step('plan', 'plan', 108, 113, { stepOp: 'AI_GATEWAY' }),
        group('search', 'search tool', [
          step('query', 'query', 113, 115, { status: 'FAILED' }),
          step('query', 'query', 117, 119, { attempts: 1 }),
          step('backoff', 'backoff', 119, 122, { stepOp: 'SLEEP' }),
          step('summarise', 'summarise', 122, 126),
        ]),
        step('approval', 'approval', 126, 132, { stepOp: 'WAIT_FOR_EVENT' }),
      ]),
    ]),
  ],
  endedAt: at(133),
  isRoot: true,
  isUserland: false,
  name: 'ci',
  outputID: null,
  queuedAt: at(0),
  scheduledAt: at(0),
  startedAt: at(0),
  spanID: 'run',
  status: 'COMPLETED',
  stepID: null,
  stepInfo: null,
  stepOp: null,
  stepType: null,
  userlandSpan: null,
  metadata: [],
};
