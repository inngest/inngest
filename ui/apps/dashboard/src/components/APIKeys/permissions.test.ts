import { describe, expect, it } from 'vitest';

import { canManageAPIKeys } from './permissions';

describe('canManageAPIKeys', () => {
  it('allows Clerk organization admins', () => {
    expect(
      canManageAPIKeys({
        isMarketplace: false,
        organizationRole: 'org:admin',
      }),
    ).toBe(true);
  });

  it('allows marketplace users without a Clerk organization role', () => {
    expect(
      canManageAPIKeys({
        isMarketplace: true,
      }),
    ).toBe(true);
  });

  it('denies non-admin Clerk organization members', () => {
    expect(
      canManageAPIKeys({
        isMarketplace: false,
        organizationRole: 'org:member',
      }),
    ).toBe(false);
  });
});
