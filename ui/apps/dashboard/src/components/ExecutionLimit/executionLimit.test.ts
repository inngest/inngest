import { describe, expect, it } from 'vitest';

import { isExecutionCapped, legacyExecutionCap } from './executionLimit';

const atLimit = {
  usage: 50_000,
  limit: 50_000,
  enforced: true,
  overageAllowed: false,
};

describe('isExecutionCapped', () => {
  it('caps once usage reaches the limit', () => {
    expect(isExecutionCapped(atLimit)).toBe(true);
    expect(isExecutionCapped({ ...atLimit, usage: 49_999 })).toBe(false);
  });

  it('never caps while enforcement is off', () => {
    expect(isExecutionCapped({ ...atLimit, enforced: false })).toBe(false);
  });

  it('never caps accounts billed for overage', () => {
    expect(isExecutionCapped({ ...atLimit, overageAllowed: true })).toBe(false);
  });
});

describe('legacyExecutionCap', () => {
  const legacyAtLimit = { usage: 50_000, limit: 50_000, overageAllowed: false };
  const cappedBy = (executions: typeof legacyAtLimit) => {
    const cap = legacyExecutionCap(executions);
    return cap && isExecutionCapped(cap);
  };

  it('treats unlimited plans as having no cap', () => {
    expect(legacyExecutionCap({ ...legacyAtLimit, limit: null })).toBeNull();
  });

  it('caps at the limit as if enforced', () => {
    expect(cappedBy(legacyAtLimit)).toBe(true);
    expect(cappedBy({ ...legacyAtLimit, usage: 49_999 })).toBe(false);
  });

  it('never caps accounts billed for overage', () => {
    expect(cappedBy({ ...legacyAtLimit, overageAllowed: true })).toBe(false);
  });
});
