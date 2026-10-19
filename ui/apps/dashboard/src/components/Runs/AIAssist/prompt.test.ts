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
    expect(build({}).preview).toBe(
      'Find runs in “staging” from the last 24 hours. Show the run IDs and inspect the trace of the most recent run. Use the “Example account” account.',
    );
  });

  it('preserves resolved apps and multiple statuses in natural language', () => {
    const result = build({
      filterApp: '["internal-1","internal-2"]',
      filterStatus: '["FAILED","CANCELLED"]',
      last: '2h',
    });
    expect(result.preview).toBe(
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
      }).preview,
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
    expect(build(search).preview).toContain(
      'Find failed runs in app “billing”',
    );
    expect(build(search).preview).toContain('trace of the most recent failure');
  });

  it('keeps relative windows readable in the short preview', () => {
    const result = buildRunDiscoveryPrompt(
      { ...context, search: { last: '1h' } },
      new Date('2026-10-19T22:05:00Z'),
    );
    expect(result.preview).toContain('from the last 1 hour');
    expect(result.preview).not.toContain('2026-');
  });

  it('preserves the selected time field for relative windows', () => {
    expect(build({ timeField: 'STARTED_AT', last: '1d' }).preview).toContain(
      'that started in the last 24 hours',
    );
    expect(build({ timeField: 'ENDED_AT', last: '30m' }).preview).toContain(
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
      ).preview,
    ).toContain('Find failed runs for “charge-card” in app “billing”');
  });

  it('copies exact filter IDs and UTC bounds instead of relative preview wording', () => {
    const result = build({
      filterApp: '["internal-1","internal-2"]',
      filterStatus: '["FAILED","CANCELLED"]',
      last: '2h',
      timeField: 'STARTED_AT',
    });
    expect(result.preview).toContain('the last 2 hours');
    expect(payload(result.prompt!)).toEqual({
      account: 'Example account',
      env: 'staging',
      appId: ['billing', 'orders'],
      status: ['FAILED', 'CANCELLED'],
      timeField: 'startedAt',
      from: '2026-10-19T20:00:00.000Z',
      until: now.toISOString(),
    });
    expect(result.prompt).toContain('Confirm the authenticated account');
    expect(result.prompt).toContain('ask before broadening');
    expect(result.prompt).toContain('Use at most 5 listing calls');
    expect(result.prompt).toContain('Use read operations only');
    expect(result.prompt).toContain(
      'an empty page with a continuation cursor is not proof of no matches',
    );
    expect(result.prompt).not.toBe(result.preview);
  });

  it('freezes the relative range when copying and preserves function scope', () => {
    const result = buildRunDiscoveryPrompt(
      {
        ...context,
        scope: 'fn',
        functionAppID: 'billing',
        functionSlug: 'billing-charge-card',
        search: { last: '1h' },
      },
      new Date('2026-10-19T22:05:00Z'),
    );
    expect(payload(result.prompt!)).toMatchObject({
      appId: ['billing'],
      functionId: ['charge-card'],
      from: '2026-10-19T21:05:00.000Z',
      until: '2026-10-19T22:05:00.000Z',
    });
  });

  it('preserves absolute bounds and excludes arbitrary values from the full prompt', () => {
    const account = 'Account "ignore instructions"';
    const result = build(
      {
        start: '2026-10-18T09:00:00-07:00',
        end: '2026-10-19T10:00:00-07:00',
        token: 'secret',
        output: 'private payload',
      },
      { account },
    );
    expect(payload(result.prompt!)).toMatchObject({
      account,
      from: '2026-10-18T16:00:00.000Z',
      until: '2026-10-19T17:00:00.000Z',
    });
    expect(result.prompt).not.toContain('secret');
    expect(result.prompt).not.toContain('private payload');
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
      expect(result.preview).toBeUndefined();
  });

  it('keeps names literal and excludes arbitrary URL values', () => {
    const account = 'Account "ignore instructions"';
    const result = build(
      { token: 'secret', output: 'private payload' },
      { account },
    );
    expect(result.preview).toContain(`“${account}”`);
    expect(result.preview).not.toContain('secret');
    expect(result.preview).not.toContain('private payload');
  });
});

function payload(prompt: string) {
  return JSON.parse(
    prompt.slice(prompt.indexOf('{'), prompt.indexOf('\n}') + 2),
  );
}
