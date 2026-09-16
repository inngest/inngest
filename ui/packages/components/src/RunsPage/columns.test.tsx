import { describe, expect, it } from 'vitest';

import { getDeferredParentLabel } from './columns';
import type { Run } from './types';

const run = {} as Run;

describe('getDeferredParentLabel', () => {
  it('falls back from name to slug to unavailable', () => {
    expect(
      getDeferredParentLabel({
        ...run,
        deferredFrom: [{ function: { name: 'Parent', slug: 'parent' } }],
      })
    ).toBe('Parent');
    expect(
      getDeferredParentLabel({
        ...run,
        deferredFrom: [{ function: { name: '', slug: 'parent' } }],
      })
    ).toBe('parent');
    expect(getDeferredParentLabel(run)).toBe('Parent function unavailable');
  });
});
