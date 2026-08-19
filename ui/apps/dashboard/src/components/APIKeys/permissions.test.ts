import { describe, expect, it } from 'vitest';

import { Marketplace } from '@/gql/graphql';
import { canManageAPIKeys } from './permissions';

describe('canManageAPIKeys', () => {
  it('allows Clerk organization admins', () => {
    expect(
      canManageAPIKeys({
        marketplace: null,
        organizationRole: 'org:admin',
      }),
    ).toBe(true);
  });

  it.each([Marketplace.Vercel, Marketplace.DigitalOcean])(
    'allows %s marketplace users without a Clerk organization role',
    (marketplace) => {
      expect(canManageAPIKeys({ marketplace })).toBe(true);
    },
  );

  it.each([Marketplace.Aws, Marketplace.Partner])(
    'does not infer admin access for %s accounts',
    (marketplace) => {
      expect(canManageAPIKeys({ marketplace })).toBe(false);
    },
  );

  it('denies non-admin Clerk organization members', () => {
    expect(
      canManageAPIKeys({
        marketplace: null,
        organizationRole: 'org:member',
      }),
    ).toBe(false);
  });
});
