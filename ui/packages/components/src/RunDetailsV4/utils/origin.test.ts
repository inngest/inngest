import { describe, expect, it } from 'vitest';

import { isInngestOrigin } from './origin';

describe('isInngestOrigin', () => {
  it.each([
    ['inngest@3.44.0', true],
    ['inngest', true],
    ['@inngest/ci@0.1.0', true],
    ['@inngest/ci', true],
    ['@inngest/agent-kit@0.9.0-beta.1', true],
    ['my-lib@1.0.0', false],
    ['@acme/inngest@1.0.0', false],
    ['inngest-plus@1.0.0', false],
    ['@inngestx/ci@1.0.0', false],
    ['', false],
    [null, false],
    [undefined, false],
  ])('%s → %s', (origin, expected) => {
    expect(isInngestOrigin(origin)).toBe(expected);
  });
});
