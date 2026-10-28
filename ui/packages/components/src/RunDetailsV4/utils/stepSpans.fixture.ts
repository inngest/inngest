/**
 * A small run trace shaped like the API's output for an `@inngest/ci` run:
 * two jobs on their own sandboxes (`e2e` also starts an extra `api` sandbox),
 * `$ …` command groups holding the steps CI runs, an ordinary step, and an
 * agent → tool nest. CI marks its own work with its origin; the user's own
 * rows (jobs, commands, `api`) carry none.
 */

import type { SandboxMetadata } from '../../generated';
import type { Trace } from '../types';

const RUN_START = Date.parse('2026-10-01T12:00:00Z');
const at = (secs: number) => new Date(RUN_START + secs * 1000).toISOString();

export const SANDBOX_A = { sandbox_id: 'sb_0a1f', sandbox_name: 'ci-01JB7Q2XKZ-base' };
export const SANDBOX_B = { sandbox_id: 'sb_7c3e', sandbox_name: 'ci-01JB7Q2XKZ-e2e' };
export const SANDBOX_C = { sandbox_id: 'sb_9d42', sandbox_name: 'ci-01JB7Q2XKZ-api' };

const CI = '@inngest/ci@0.1.0';

const base: Trace = {
  attempts: 0,
  endedAt: null,
  isRoot: false,
  isUserland: false,
  name: '',
  outputID: null,
  queuedAt: at(0),
  scheduledAt: null,
  startedAt: null,
  spanID: '',
  status: 'COMPLETED',
  stepID: null,
  stepInfo: null,
  stepOp: null,
  stepType: null,
  userlandSpan: null,
  metadata: [],
};

function step(
  name: string,
  from: number,
  to: number,
  extra: Partial<Trace> & { sandbox?: Partial<SandboxMetadata> } = {}
): Trace {
  const { sandbox, ...trace } = extra;

  return {
    ...base,
    name,
    spanID: `span-${name}`,
    stepID: name,
    stepOp: 'RUN',
    stepType: 'RUN',
    queuedAt: at(from),
    startedAt: at(from),
    endedAt: at(to),
    metadata: sandbox
      ? [
          {
            kind: 'inngest.sandbox',
            scope: 'step',
            updatedAt: at(to),
            values: { version: 1, action: 'exec', method: 'commands.run', ...sandbox },
          },
        ]
      : [],
    ...trace,
  };
}

/** A group as the loader builds it: children by queue time, times from them */
function group(name: string, children: Trace[], groupKind?: string, origin?: string): Trace {
  return {
    ...base,
    name,
    spanID: `span:${name}`,
    stepType: 'SPAN_GROUP',
    childrenSpans: children,
    queuedAt: children[0]!.queuedAt,
    startedAt: children[0]!.startedAt,
    endedAt: children[children.length - 1]!.endedAt,
    groupKind: groupKind ?? null,
    origin: origin ?? null,
  };
}

/** A `$ …` group: the user's command, with the steps CI runs inside */
function command(text: string, from: number, to: number, sandbox = SANDBOX_A): Trace {
  const mid = (from + to) / 2;

  return group(`$ ${text}`, [
    step(`${text} start`, from, mid, {
      origin: CI,
      sandbox: { ...sandbox, command_display: text },
    }),
    step(`${text} output`, mid, to, { origin: CI, sandbox: { ...sandbox, command_display: text } }),
  ]);
}

function startSandbox(from: number, to: number, sandbox: typeof SANDBOX_A): Trace {
  return group(
    'Start sandbox',
    [step('Create sandbox', from, to, { origin: CI, sandbox: { ...sandbox, action: 'create' } })],
    undefined,
    CI
  );
}

export const stepSpansTrace: Trace = {
  ...base,
  isRoot: true,
  name: 'ci',
  spanID: 'run',
  queuedAt: at(0),
  startedAt: at(0),
  endedAt: at(60),
  childrenSpans: [
    group('GitHub', [step('Create check', 0, 1, { origin: CI })], undefined, CI),
    group(
      'base',
      [
        startSandbox(0, 6, SANDBOX_A),
        command('pnpm lint', 6, 14),
        command('pnpm test', 14, 22),
        group(
          'Snapshot sandbox',
          [
            step('Create snapshot', 22, 25, {
              origin: 'inngest@3.44.0',
              sandbox: { ...SANDBOX_A, action: 'snapshot.create' },
            }),
            step('Wait for snapshot', 25, 30, { origin: 'inngest@3.44.0', stepOp: 'SLEEP' }),
          ],
          undefined,
          CI
        ),
      ],
      'job'
    ),
    group(
      'e2e',
      [
        startSandbox(30, 34, SANDBOX_B),
        group('api', [startSandbox(34, 38, SANDBOX_C), command('pnpm api', 38, 40, SANDBOX_C)]),
        command('pnpm e2e', 40, 44, SANDBOX_B),
        group(
          'Save sandbox',
          [
            step('Pause sandbox', 44, 45, {
              origin: CI,
              sandbox: { ...SANDBOX_B, action: 'pause' },
            }),
          ],
          undefined,
          CI
        ),
      ],
      'job'
    ),
    step('notify', 46, 48),
    group('Research network', [
      group(
        'Research agent',
        [
          step('plan', 48, 52),
          group('search tool', [step('query', 52, 54), step('summarise', 54, 56)], 'tool'),
        ],
        'agent'
      ),
    ]),
    step('Clean up sandboxes', 56, 58, { origin: CI }),
  ],
};
