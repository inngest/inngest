import { describe, expect, it } from 'vitest';

import { isDimmed, isInngestOrigin } from './origin';

describe('isInngestOrigin', () => {
  it.each([
    ['inngest@3.44.0', true],
    ['@inngest/ci@0.1.0', true],
    ['@inngest/ci', true],
    ['@inngest/agent-kit@0.9.0-beta.1', true],
    ['my-lib@1.0.0', false],
    ['@acme/inngest@1.0.0', false],
    ['', false],
    [null, false],
  ])('%s → %s', (origin, expected) => {
    expect(isInngestOrigin(origin)).toBe(expected);
  });
});

describe('isDimmed', () => {
  it('dims what Inngest added unless it failed', () => {
    expect(isDimmed('@inngest/ci@0.1.0', 'COMPLETED')).toBe(true);
    expect(isDimmed('@inngest/ci@0.1.0', 'RUNNING')).toBe(true);
    expect(isDimmed('@inngest/ci@0.1.0', 'FAILED')).toBe(false);
    expect(isDimmed('my-lib@1.0.0', 'COMPLETED')).toBe(false);
    expect(isDimmed(undefined, 'COMPLETED')).toBe(false);
  });
});
