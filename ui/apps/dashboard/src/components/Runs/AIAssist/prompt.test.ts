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
  it('uses a precise default window', () => {
    expect(payload(build({}).prompt!)).toMatchObject({
      timeField: 'queuedAt',
      from: '2026-10-18T22:00:00.000Z',
      until: now.toISOString(),
    });
  });

  it.each([
    ['QUEUED_AT', 'queuedAt'],
    ['STARTED_AT', 'startedAt'],
    ['ENDED_AT', 'endedAt'],
  ])('preserves %s and absolute UTC boundaries', (field, apiField) => {
    expect(
      payload(
        build({
          timeField: field,
          start: '2026-10-18T09:00:00-07:00',
          end: '2026-10-19T10:00:00-07:00',
        }).prompt!,
      ),
    ).toMatchObject({
      timeField: apiField,
      from: '2026-10-18T16:00:00.000Z',
      until: '2026-10-19T17:00:00.000Z',
    });
  });

  it('reads selections after the actual router URL round trip', () => {
    const search = defaultParseSearch(
      defaultStringifySearch({
        filterApp: '["internal-1"]',
        filterStatus: '["FAILED"]',
      }),
    );
    expect(payload(build(search).prompt!)).toMatchObject({
      appId: ['billing'],
      status: ['FAILED'],
    });
  });

  it('includes exact filter IDs, UTC bounds, and investigation instructions', () => {
    const result = build({
      filterApp: '["internal-1","internal-2"]',
      filterStatus: '["FAILED","CANCELLED"]',
      last: '2h',
      timeField: 'STARTED_AT',
    });
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
  });

  it('freezes the relative range at the snapshot time and preserves function scope', () => {
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
      expect(result.prompt).toBeUndefined();
  });
});

function payload(prompt: string) {
  return JSON.parse(
    prompt.slice(prompt.indexOf('{'), prompt.indexOf('\n}') + 2),
  );
}
