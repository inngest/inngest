/**
 * A hand-built run trace shaped like the API's output for a run of
 * `@inngest/ci`, for tests and for previewing the timeline. It mirrors the span
 * tree CI emits, as drawn by its `spans.test.ts`:
 *
 * - a `GitHub` span of check updates, and `Clean up sandboxes` at the end
 * - a `base` job on sandbox A: `Start sandbox`, `$ …` command spans, a command
 *   retried in `Attempt 1`/`Attempt 2` spans, a background `$ pnpm dev` span
 *   re-entered to kill it at the end, and a `Save sandbox` span holding the
 *   `Snapshot sandbox` span, whose steps are the SDK's
 * - an `e2e` job on sandbox B that uses an extra sandbox C in its own `api`
 *   span, with its commands inside it
 * - an ordinary step outside any group
 * - an agent → tool → steps nest three groups deep, with a retried step
 *
 * Groups carry the kinds CI and agents give them (`job`, `agent`, `tool`). CI
 * marks only its own work with its origin (`@inngest/ci@0.1.0`), and the SDK
 * the snapshot's steps (`inngest@3.44.0`). Jobs, `$ …` commands, `api` and the
 * user's own rows carry none, because the user wrote them.
 */

import type { SandboxMetadata } from '../../generated';
import type { Trace } from '../types';

const RUN_START = Date.parse('2026-10-01T12:00:00Z');
const at = (secs: number) => new Date(RUN_START + secs * 1000).toISOString();

export const SANDBOX_A = { sandbox_id: 'sb_0a1f', sandbox_name: 'ci-01JB7Q2XKZ-base' };
export const SANDBOX_B = { sandbox_id: 'sb_7c3e', sandbox_name: 'ci-01JB7Q2XKZ-e2e' };
export const SANDBOX_C = { sandbox_id: 'sb_9d42', sandbox_name: 'ci-01JB7Q2XKZ-api' };

export const CI_ORIGIN = '@inngest/ci@0.1.0';
export const SDK_ORIGIN = 'inngest@3.44.0';

/** Marks a row as added by a library on the user's behalf */
function addedBy(origin: string, trace: Trace): Trace {
  return { ...trace, origin };
}

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

/** A step CI runs on the user's behalf */
function ciStep(
  id: string,
  name: string,
  from: number,
  to: number | null,
  options: StepOptions = {}
): Trace {
  return addedBy(CI_ORIGIN, step(id, name, from, to, options));
}

const exec = (
  command: string,
  exitCode: number,
  sandbox = SANDBOX_A
): Omit<SandboxMetadata, 'version'> => ({
  action: 'exec',
  method: 'commands.run',
  command_display: command,
  exit_code: exitCode,
  ...sandbox,
});

/** A group as the loader builds it: children by queue time, times and status from them */
function group(
  id: string,
  name: string,
  children: Trace[],
  groupKind?: string,
  origin?: string
): Trace {
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
    groupKind: groupKind ?? null,
    origin: origin ?? null,
    userlandSpan: null,
    metadata: [],
  };
}

/** A span CI opens for its own work */
function ciGroup(id: string, name: string, children: Trace[]): Trace {
  return group(id, name, children, undefined, CI_ORIGIN);
}

/** The steps CI runs for a captured command, spread evenly over `from`..`to` */
function commandSteps(
  id: string,
  command: string,
  from: number,
  to: number,
  { sandbox = SANDBOX_A, exitCode = 0 } = {}
): Trace[] {
  const quarter = (to - from) / 4;
  const failed = exitCode !== 0;

  return [
    ciStep(`${id}-start`, 'Start process', from, from + quarter, {
      sandbox: { ...exec(command, 0, sandbox), action: 'process.start', process_id: `p_${id}` },
    }),
    ciStep(`${id}-wait`, 'Wait 1s', from + quarter, from + 2 * quarter, { stepOp: 'SLEEP' }),
    ciStep(`${id}-poll`, 'Poll process', from + 2 * quarter, from + 3 * quarter, {
      sandbox: { ...exec(command, 0, sandbox), action: 'process.status', process_id: `p_${id}` },
    }),
    ciStep(`${id}-output`, 'Read output', from + 3 * quarter, to - (failed ? 0.5 : 0), {
      sandbox: { ...exec(command, exitCode, sandbox), action: 'process.output' },
    }),
    ...(failed
      ? [
          ciStep(`${id}-exit`, `Exited with code ${exitCode}`, to - 0.5, to, {
            status: 'FAILED',
            sandbox: exec(command, exitCode, sandbox),
          }),
        ]
      : []),
  ];
}

/** A `$ …` span: the user's command, with CI's steps inside */
function command(id: string, text: string, from: number, to: number, sandbox = SANDBOX_A): Trace {
  return group(id, `$ ${text}`, commandSteps(id, text, from, to, { sandbox }));
}

/** A `Start sandbox` span, with the steps that create it and prepare its workspace */
function startSandbox(
  id: string,
  name: string,
  from: number,
  to: number,
  sandbox: typeof SANDBOX_A
) {
  const mid = (from + to) / 2;

  return ciGroup(id, name, [
    ciStep(`${id}-create`, 'Create sandbox', from, mid, {
      sandbox: { action: 'create', method: 'create', ...sandbox },
    }),
    ciStep(`${id}-setup`, 'Prepare workspace', mid, to, {
      sandbox: { ...exec('mkdir -p /workspace', 0, sandbox) },
    }),
  ]);
}

export const stepSpansTrace: Trace = {
  attempts: 0,
  childrenSpans: [
    ciGroup('github', 'GitHub', [
      ciStep('gh-create', 'Create check: pr', 0, 1),
      ciStep('gh-base-start', 'Report base: started', 1, 2),
      ciStep('gh-e2e-start', 'Report e2e: started', 90, 91),
      ciStep('gh-base-pass', 'Report base: passed', 105, 106),
      ciStep('gh-e2e-pass', 'Report e2e: passed', 106, 107),
      ciStep('gh-complete', 'Complete check: pr', 132, 133),
    ]),
    group(
      'base',
      'base',
      [
        startSandbox('base-machine', 'Start sandbox', 0, 6, SANDBOX_A),
        command('setup', 'pnpm install --frozen-lockfile', 6, 14),
        command('lint', 'pnpm lint', 14, 22),
        group('dev-server', '$ pnpm dev', [
          ...commandSteps('dev', 'pnpm dev', 22, 30),
          ciStep('dev-kill', 'Kill process', 104, 106, {
            sandbox: { ...exec('pnpm dev', 143), action: 'process.kill', process_id: 'p_dev' },
          }),
        ]),
        group('test', '$ pnpm test', [
          ciGroup(
            'test-1',
            'Attempt 1',
            commandSteps('test-a1', 'pnpm test', 30, 52, { exitCode: 1 })
          ),
          ciStep('retry-check', 'Wait 6s', 52, 58, { stepOp: 'SLEEP' }),
          ciGroup('test-2', 'Attempt 2', commandSteps('test-a2', 'pnpm test', 58, 78)),
        ]),
        ciGroup('save', 'Save sandbox', [
          ciStep('pause', 'Pause sandbox', 78, 79),
          ciStep('resume', 'Resume sandbox', 79, 80),
          ciStep('record', 'Record snapshot contents', 80, 81),
          group(
            'snapshot',
            'Snapshot sandbox',
            [
              addedBy(
                SDK_ORIGIN,
                step('snap-create', 'Create snapshot', 81, 84, {
                  sandbox: { action: 'snapshot.create', method: 'snapshot', ...SANDBOX_A },
                })
              ),
              addedBy(
                SDK_ORIGIN,
                step('snap-ready', 'Wait for snapshot', 84, 90, {
                  stepOp: 'SLEEP',
                  sandbox: {
                    action: 'snapshot.waitUntilReady',
                    method: 'snapshot',
                    snapshot_id: 'snap_1',
                  },
                })
              ),
            ],
            undefined,
            CI_ORIGIN
          ),
        ]),
      ],
      'job'
    ),
    group(
      'e2e',
      'e2e',
      [
        startSandbox('e2e-machine', 'Start sandbox from base', 90, 94, SANDBOX_B),
        group('api', 'api', [
          startSandbox('api-machine', 'Start sandbox', 94, 98, SANDBOX_C),
          command('api-cmd', 'pnpm api', 98, 100, SANDBOX_C),
        ]),
        command('e2e', 'pnpm e2e', 100, 104, SANDBOX_B),
        ciGroup('e2e-save', 'Save sandbox', [
          ciStep('e2e-pause', 'Pause sandbox', 104, 105, {
            sandbox: { ...exec('pause', 0, SANDBOX_B), action: 'pause' },
          }),
        ]),
      ],
      'job'
    ),
    step('notify', 'notify', 106, 108),
    group('network', 'Research network', [
      group(
        'agent',
        'Research agent',
        [
          step('plan', 'plan', 108, 113, { stepOp: 'AI_GATEWAY' }),
          group(
            'search',
            'search tool',
            [
              step('query', 'query', 113, 115, { status: 'FAILED' }),
              step('query', 'query', 117, 119, { attempts: 1 }),
              step('backoff', 'backoff', 119, 122, { stepOp: 'SLEEP' }),
              step('summarise', 'summarise', 122, 126),
            ],
            'tool'
          ),
          step('approval', 'approval', 126, 132, { stepOp: 'WAIT_FOR_EVENT' }),
        ],
        'agent'
      ),
    ]),
    ciStep('cleanup', 'Clean up sandboxes', 133, 135),
  ],
  endedAt: at(135),
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
