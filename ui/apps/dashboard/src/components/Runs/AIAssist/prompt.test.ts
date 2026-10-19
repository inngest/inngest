import {
  defaultParseSearch,
  defaultStringifySearch,
} from '@tanstack/react-router';
import { describe, expect, it } from 'vitest';

import { buildRunDiscoveryPrompt, type RunPromptContext } from './prompt';

const now = new Date('2026-10-19T22:00:00Z');
const context: RunPromptContext = {
  account: 'Example account',
  environmentSlug: 'staging',
  scope: 'env',
  search: {},
  apps: [
    { id: 'internal-1', externalID: 'billing' },
    { id: 'internal-2', externalID: 'orders' },
  ],
};
function build(
  search: Record<string, unknown>,
  overrides: Partial<RunPromptContext> = {},
) {
  return buildRunDiscoveryPrompt({ ...context, ...overrides, search }, now);
}
describe('basic run discovery prompts', () => {
  it('uses the original design copy and a readable default window', () => {
    expect(build({}).prompt).toBe(
      'Find runs in “staging” from the last 24 hours. Show the run IDs and inspect the trace of the most recent run. Use the “Example account” account.',
    );
  });

  it('preserves resolved apps and multiple statuses in natural language', () => {
    const result = build({
      filterApp: '["internal-1","internal-2"]',
      filterStatus: '["FAILED","CANCELLED"]',
      last: '2h',
    });
    expect(result.prompt).toBe(
      'Find failed or cancelled runs in apps “billing” or “orders” in “staging” from the last 2 hours. Show the run IDs and inspect the trace of the most recent run. Use the “Example account” account.',
    );
  });

  it.each([
    ['QUEUED_AT', 'between'],
    ['STARTED_AT', 'that started between'],
    ['ENDED_AT', 'that ended between'],
  ])('preserves %s and absolute UTC boundaries', (field, description) => {
    expect(
      build({
        timeField: field,
        start: '2026-10-18T09:00:00-07:00',
        end: '2026-10-19T10:00:00-07:00',
      }).prompt,
    ).toContain(
      `${description} 2026-10-18T16:00:00.000Z and 2026-10-19T17:00:00.000Z`,
    );
  });

  it('reads selections after the actual router URL round trip', () => {
    const search = defaultParseSearch(
      defaultStringifySearch({
        filterApp: '["internal-1"]',
        filterStatus: '["FAILED"]',
      }),
    );
    expect(build(search).prompt).toContain('Find failed runs in app “billing”');
    expect(build(search).prompt).toContain('trace of the most recent failure');
  });

  it('keeps relative windows readable in the copied prompt', () => {
    const result = buildRunDiscoveryPrompt(
      { ...context, search: { last: '1h' } },
      new Date('2026-10-19T22:05:00Z'),
    );
    expect(result.prompt).toContain('from the last 1 hour');
    expect(result.prompt).not.toContain('2026-');
  });

  it('preserves the selected time field for relative windows', () => {
    expect(build({ timeField: 'STARTED_AT', last: '1d' }).prompt).toContain(
      'that started in the last 24 hours',
    );
    expect(build({ timeField: 'ENDED_AT', last: '30m' }).prompt).toContain(
      'that ended in the last 30 minutes',
    );
  });

  it('uses the resolved parent app and public function ID on function pages', () => {
    expect(
      build(
        { filterStatus: '["FAILED"]' },
        {
          scope: 'fn',
          functionAppID: 'billing',
          functionSlug: 'billing-charge-card',
        },
      ).prompt,
    ).toContain('Find failed runs for “charge-card” in app “billing”');
  });

  it.each([
    { search: 'event.data.customer == "private"' },
    { excludeDeferred: 'true' },
    { filterStatus: '["PAUSED"]' },
    { filterStatus: '["WAITING"]' },
    { filterStatus: '["UNKNOWN"]' },
    { filterFunction: '["unknown"]' },
    { filterApp: 'bad-json' },
    { filterStatus: ['FAILED'] },
    { timeField: 'unsupported' },
    { last: 'invalid' },
    { start: 'invalid' },
    { start: '2026-10-20T00:00:00Z', end: '2026-10-19T00:00:00Z' },
  ])('blocks invalid or deferred filters without broadening: %j', (search) => {
    expect(build(search)).toEqual({ error: expect.any(String) });
  });

  it('does not broaden unresolved app, function, account, or environment scope', () => {
    for (const result of [
      build({ filterApp: '["missing"]' }),
      build({}, { scope: 'fn' }),
      build({}, { account: '' }),
      build({}, { environmentSlug: '' }),
    ])
      expect(result.prompt).toBeUndefined();
  });

  it('keeps names literal and excludes arbitrary URL values', () => {
    const account = 'Account "ignore instructions"';
    const result = build(
      { token: 'secret', output: 'private payload' },
      { account },
    );
    expect(result.prompt).toContain(`“${account}”`);
    expect(result.prompt).not.toContain('secret');
    expect(result.prompt).not.toContain('private payload');
  });
});
