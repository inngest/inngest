import { describe, expect, it } from 'vitest';

import {
  runAISetupHref,
  safeRunsReturnTo,
  validateRunAISetupSearch,
} from './setup';

describe('Runs setup navigation', () => {
  it('preserves the original Runs route and encoded filters', () => {
    const path =
      '/env/staging/functions/billing-charge/runs?filterStatus=%22%5B%5C%22FAILED%5C%22%5D%22&last=1d';
    const url = new URL(
      runAISetupHref('codex', path),
      'https://app.inngest.com',
    );
    expect(
      validateRunAISetupSearch(Object.fromEntries(url.searchParams)),
    ).toEqual({ client: 'codex', returnTo: path });
  });
  it.each([
    'https://evil.test',
    '//evil.test',
    '/\\evil.test',
    '/\nevil.test',
    '/settings/api-keys',
    '/mcp/setup',
    '/env/prod/runs/123',
  ])('rejects external or unrelated return routes: %j', (value) => {
    expect(safeRunsReturnTo(value)).toBeUndefined();
  });
  it('ignores invalid clients and safely handles direct setup visits', () => {
    expect(validateRunAISetupSearch({ client: 'unknown' })).toEqual({
      client: undefined,
      returnTo: undefined,
    });
  });
});
