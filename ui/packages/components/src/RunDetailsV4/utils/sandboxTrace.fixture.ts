/**
 * A CI-like run with two sandboxes, shaped like the scenario in
 * docs/sandbox-traces/README.md. Used by tests and the Timeline story.
 *
 * - `ci-build` is created and destroyed by the run.
 * - `ci-e2e` already existed: the run picks it up with `get` and leaves it.
 * - `test` is a CI command with no statement step of its own: a start, three
 *   sleeps, three polls and an output fetch, all `role: "internal"`.
 * - `snapshot-build` is a snapshot statement plus its readiness wait.
 * - `notify-start` is an ordinary step.run with no sandbox metadata.
 */

import type { SandboxMetadata } from '../../generated';
import type { Trace } from '../types';

const T0 = Date.parse('2026-10-01T12:00:00.000Z');

export const BUILD_SANDBOX_ID = '7f3a2c1e-4b5d-4e6f-8a9b-0c1d2e3f4a5b';
export const E2E_SANDBOX_ID = '91c2d3e4-f5a6-4b7c-8d9e-0f1a2b3c4d5e';
export const TEST_STATEMENT_ID = 'c0ffee000000000000000000000000000000test';
export const SNAPSHOT_STATEMENT_ID = 'c0ffee00000000000000000000000000snapshot';

function at(seconds: number): string {
  return new Date(T0 + seconds * 1000).toISOString();
}

let spanCounter = 0;

/** A run child: one attempt of a step, with sandbox metadata when given */
export function sandboxStep({
  name,
  stepID,
  start,
  end,
  status = 'COMPLETED',
  stepOp = 'RUN',
  attempt = 0,
  sandbox,
}: {
  name: string;
  stepID: string;
  start: number;
  end: number | null;
  status?: string;
  stepOp?: string;
  attempt?: number;
  sandbox?: Partial<SandboxMetadata>;
}): Trace {
  spanCounter += 1;
  const isSleep = stepOp === 'SLEEP';
  return {
    attempts: attempt,
    endedAt: end === null ? null : at(end),
    isRoot: false,
    isUserland: false,
    name,
    outputID: isSleep || end === null ? null : `output-${stepID}-${attempt}`,
    queuedAt: at(start),
    scheduledAt: at(start),
    spanID: `span-${spanCounter}`,
    stepID,
    startedAt: at(start + 0.05),
    status,
    stepInfo: isSleep ? { sleepUntil: end === null ? at(start + 30) : at(end) } : { type: null },
    stepOp,
    userlandSpan: null,
    metadata: sandbox
      ? [
          {
            scope: 'step_attempt',
            kind: 'inngest.sandbox',
            updatedAt: end === null ? at(start) : at(end),
            values: {
              version: 1,
              role: 'statement',
              statement_id: stepID,
              ...sandbox,
            } as SandboxMetadata,
          },
        ]
      : undefined,
  };
}

const build = { sandbox_id: BUILD_SANDBOX_ID, sandbox_name: 'ci-build' };
const e2e = { sandbox_id: E2E_SANDBOX_ID, sandbox_name: 'ci-e2e' };

/** Internal metadata for one step of the CI `test` command */
function testStep(action: string, extra: Partial<SandboxMetadata> = {}): Partial<SandboxMetadata> {
  return {
    ...build,
    action,
    statement: 'commands.run',
    statement_id: TEST_STATEMENT_ID,
    statement_name: 'test',
    role: 'internal',
    command: ['/bin/sh', '-c', 'pnpm test'],
    command_display: 'pnpm test',
    ...extra,
  };
}

/** The CI `test` command's steps (start, sleeps, polls, output) */
export function ciTestSteps(): Trace[] {
  const process = { process_id: 'proc-test-1' };
  return [
    sandboxStep({
      name: 'test › start',
      stepID: 'a1000000000000000000000000000000000start',
      start: 38,
      end: 38.5,
      sandbox: testStep('process.start', { ...process, process_state: 'RUNNING' }),
    }),
    sandboxStep({
      name: 'test › wait #1',
      stepID: 'a10000000000000000000000000000000wait01',
      start: 38.5,
      end: 53.5,
      stepOp: 'SLEEP',
      sandbox: testStep('sleep'),
    }),
    sandboxStep({
      name: 'test › check #1',
      stepID: 'a1000000000000000000000000000000check01',
      start: 53.6,
      end: 54,
      sandbox: testStep('process.get', { ...process, process_state: 'RUNNING' }),
    }),
    sandboxStep({
      name: 'test › wait #2',
      stepID: 'a10000000000000000000000000000000wait02',
      start: 54,
      end: 84,
      stepOp: 'SLEEP',
      sandbox: testStep('sleep'),
    }),
    sandboxStep({
      name: 'test › check #2',
      stepID: 'a1000000000000000000000000000000check02',
      start: 84.1,
      end: 84.5,
      sandbox: testStep('process.get', { ...process, process_state: 'RUNNING' }),
    }),
    sandboxStep({
      name: 'test › wait #3',
      stepID: 'a10000000000000000000000000000000wait03',
      start: 84.5,
      end: 114.5,
      stepOp: 'SLEEP',
      sandbox: testStep('sleep'),
    }),
    sandboxStep({
      name: 'test › check #3',
      stepID: 'a1000000000000000000000000000000check03',
      start: 114.6,
      end: 115.1,
      sandbox: testStep('process.get', { ...process, process_state: 'EXITED', exit_code: 0 }),
    }),
    sandboxStep({
      name: 'test › output',
      stepID: 'a100000000000000000000000000000000output',
      start: 115.2,
      end: 122.2,
      sandbox: testStep('process.output', process),
    }),
  ];
}

/** The `snapshot-build` statement and its readiness wait */
export function snapshotSteps(): Trace[] {
  const snapshot = { ...build, statement: 'snapshot', snapshot_id: 'snap-build-1' };
  return [
    sandboxStep({
      name: 'snapshot-build',
      stepID: SNAPSHOT_STATEMENT_ID,
      start: 122.5,
      end: 123.3,
      sandbox: { ...snapshot, action: 'snapshot.create', snapshot_status: 'CREATING' },
    }),
    sandboxStep({
      name: 'snapshot-build:wait-until-ready',
      stepID: 'b2000000000000000000000000wait-until-ready',
      start: 123.3,
      end: 143.3,
      sandbox: {
        ...snapshot,
        action: 'snapshot.waitUntilReady',
        statement_id: SNAPSHOT_STATEMENT_ID,
        role: 'internal',
        snapshot_status: 'READY',
      },
    }),
  ];
}

/** The whole run, before traceRollup */
export function sandboxCIRun(): Trace {
  const children: Trace[] = [
    sandboxStep({
      name: 'build-machine',
      stepID: 'd1000000000000000000000000build-machine',
      start: 0,
      end: 5.1,
      sandbox: { ...build, action: 'create', statement: 'create' },
    }),
    sandboxStep({
      name: 'get-e2e',
      stepID: 'd10000000000000000000000000000000get-e2e',
      start: 0.2,
      end: 0.6,
      sandbox: { ...e2e, action: 'get', statement: 'get' },
    }),
    sandboxStep({
      name: 'install',
      stepID: 'd10000000000000000000000000000000install',
      start: 6,
      end: 34.7,
      sandbox: {
        ...build,
        action: 'exec',
        statement: 'commands.run',
        command: ['/bin/sh', '-c', 'pnpm install'],
        command_display: 'pnpm install',
        exit_code: 0,
      },
    }),
    sandboxStep({
      name: 'e2e-install',
      stepID: 'd100000000000000000000000000e2e-install',
      start: 6.5,
      end: 37.7,
      sandbox: {
        ...e2e,
        action: 'exec',
        statement: 'commands.run',
        command: ['/bin/sh', '-c', 'pnpm install'],
        command_display: 'pnpm install',
        exit_code: 0,
      },
    }),
    sandboxStep({
      name: 'notify-start',
      stepID: 'd100000000000000000000000000notify-start',
      start: 38,
      end: 38.9,
    }),
    ...ciTestSteps(),
    sandboxStep({
      name: 'e2e',
      stepID: 'd1000000000000000000000000000000000000e2e',
      start: 40,
      end: 98.7,
      status: 'FAILED',
      sandbox: {
        ...e2e,
        action: 'exec',
        statement: 'commands.run',
        command: ['/bin/sh', '-c', 'pnpm e2e --shard 1/2'],
        command_display: 'pnpm e2e --shard 1/2',
        exit_code: 1,
      },
    }),
    sandboxStep({
      name: 'e2e-release',
      stepID: 'd100000000000000000000000000e2e-release',
      start: 99,
      end: 99.7,
      sandbox: {
        ...e2e,
        action: 'exec',
        statement: 'commands.run',
        command: ['rm', '-rf', '/work'],
        exit_code: 0,
      },
    }),
    ...snapshotSteps(),
    sandboxStep({
      name: 'destroy-build',
      stepID: 'd100000000000000000000000destroy-build',
      start: 143.5,
      end: 144.3,
      sandbox: { ...build, action: 'destroy', statement: 'destroy' },
    }),
  ];

  return {
    attempts: 0,
    childrenSpans: children,
    endedAt: at(145),
    isRoot: true,
    isUserland: false,
    name: 'ci/pull-request',
    outputID: 'output-run',
    queuedAt: at(0),
    scheduledAt: at(0),
    spanID: 'span-run',
    startedAt: at(0),
    status: 'FAILED',
    stepInfo: null,
    userlandSpan: null,
  };
}
