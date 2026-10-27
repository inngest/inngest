import { describe, expect, it } from 'vitest';
import {
  matchingRun,
  readAttempt,
  readCompletion,
  storageKey,
  type QuickstartRun,
} from './state';
import { createOnboardingPrompt } from './prompt';

const server = 'http://localhost:8288';
const attempt = { functionID: 'fn-1', startedAt: '2026-10-23T12:00:00.000Z' };
const run: QuickstartRun = {
  id: 'run-1',
  appID: 'app-1',
  function: { id: 'fn-1', name: 'Hello', slug: 'app-hello' },
  status: 'COMPLETED',
  queuedAt: '2026-10-23T12:00:01.000Z',
  endedAt: '2026-10-23T12:00:02.000Z',
};

describe('quickstart completion and run reconciliation', () => {
  it('scopes versioned completion to the resolved server', () => {
    const marker = {
      version: 1,
      server,
      completedAt: run.endedAt,
      appID: run.appID,
      functionID: run.function.id,
      runID: run.id,
    };
    expect(readCompletion(JSON.stringify(marker), server)).toEqual(marker);
    expect(
      readCompletion(JSON.stringify(marker), 'http://localhost:8289'),
    ).toBeNull();
    expect(storageKey(server)).not.toBe(storageKey('http://localhost:8289'));
    expect(readCompletion('{broken', server)).toBeNull();
    expect(
      readCompletion(JSON.stringify({ ...marker, version: 2 }), server),
    ).toBeNull();
  });
  it('restores only attempt metadata, excluding payloads and credentials', () => {
    expect(
      readAttempt(
        JSON.stringify({ ...attempt, payload: { secret: 'private' } }),
      ),
    ).toEqual(attempt);
    expect(
      readAttempt(JSON.stringify({ ...attempt, startedAt: 'invalid' })),
    ).toBeNull();
  });
  it('infers prior completion only from a successful retained run', () => {
    for (const status of [
      'QUEUED',
      'RUNNING',
      'FAILED',
      'CANCELLED',
      'SKIPPED',
    ]) {
      expect(matchingRun([{ ...run, status }], null, false)).toBeUndefined();
    }
    expect(matchingRun([run], null, false)).toEqual(run);
  });
  it('excludes runs before the attempt and runs from other functions', () => {
    expect(
      matchingRun(
        [{ ...run, queuedAt: '2026-10-23T11:59:59Z' }],
        attempt,
        false,
      ),
    ).toBeUndefined();
    expect(
      matchingRun(
        [{ ...run, function: { ...run.function, id: 'fn-2' } }],
        attempt,
        false,
      ),
    ).toBeUndefined();
    expect(matchingRun([run], attempt, false)).toEqual(run);
  });
  it('requires explicit selection when runs are ambiguous or paginated', () => {
    const other = { ...run, id: 'run-2' };
    expect(matchingRun([run, other], attempt, false)).toBeUndefined();
    expect(matchingRun([run], attempt, true)).toBeUndefined();
    expect(
      matchingRun([run, other], { ...attempt, runID: other.id }, true),
    ).toEqual(other);
    expect(
      matchingRun([run], { ...attempt, runID: 'missing' }, false),
    ).toBeUndefined();
  });
  it('keeps the prompt scoped to the active local endpoint and verified first run', () => {
    const prompt = createOnboardingPrompt('http://127.0.0.1:9999');
    expect(prompt).toContain('Target MCP endpoint: http://127.0.0.1:9999/mcp');
    expect(prompt).toContain('inspect that exact run until it completes');
    expect(prompt).toContain('Do not deploy to Cloud');
    expect(prompt).not.toContain('localhost:8288');
    expect(prompt).toContain(
      'claude mcp add --transport http inngest-dev http://127.0.0.1:9999/mcp',
    );
    expect(prompt).toContain(
      'codex mcp add inngest-dev --url http://127.0.0.1:9999/mcp',
    );
    expect(prompt).toContain('pause, and resume this task after reconnecting');
  });
});
