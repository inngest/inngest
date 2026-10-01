import { beforeEach, describe, expect, it, vi } from 'vitest';

import { Marketplace } from '@/gql/graphql';

import { useExecutionLimit } from './useExecutionLimit';

const mocks = vi.hoisted(() => ({
  useBooleanFlag: vi.fn(),
  useSkippableGraphQLQuery: vi.fn(),
}));

vi.mock('@/components/FeatureFlags/hooks', () => ({
  useBooleanFlag: mocks.useBooleanFlag,
}));

vi.mock('@/gql', () => ({
  graphql: (source: string) => source,
}));

vi.mock('@/utils/useGraphQLQuery', () => ({
  useSkippableGraphQLQuery: mocks.useSkippableGraphQLQuery,
}));

describe('useExecutionLimit', () => {
  let flag: { value: boolean; isReady: boolean };
  let legacyAccount: {
    id: string;
    marketplaceBillingURL: string | null;
    entitlements: {
      executions: {
        limit: number | null;
        overageAllowed: boolean;
      };
      usage: { executions: number };
    };
  };
  let enhancedAccount: {
    id: string;
    marketplace: Marketplace;
    marketplaceBillingURL: string | null;
    executionCap: {
      usage: number;
      limit: number;
      enforced: boolean;
      exceeded: boolean;
    } | null;
  };

  beforeEach(() => {
    vi.resetAllMocks();
    flag = { value: false, isReady: true };
    legacyAccount = {
      id: 'legacy-account',
      marketplaceBillingURL: null,
      entitlements: {
        executions: { limit: 50_000, overageAllowed: false },
        usage: { executions: 50_000 },
      },
    };
    enhancedAccount = {
      id: 'enhanced-account',
      marketplace: Marketplace.Vercel,
      marketplaceBillingURL: 'https://vercel.com/billing',
      executionCap: {
        usage: 50_000,
        limit: 50_000,
        enforced: false,
        exceeded: true,
      },
    };
    mocks.useBooleanFlag.mockImplementation((name: string) =>
      name === 'hobby-execution-limit-ui'
        ? flag
        : { value: true, isReady: true },
    );
    mocks.useSkippableGraphQLQuery.mockImplementation(
      ({ query, skip }: { query: string; skip: boolean }) => ({
        data: skip
          ? undefined
          : {
              account: query.includes('ExecutionLimitCheck')
                ? legacyAccount
                : enhancedAccount,
            },
      }),
    );
  });

  it('uses legacy limits when the flag is off', () => {
    flag = { value: false, isReady: true };

    expect(useExecutionLimit()).toMatchObject({
      accountID: 'legacy-account',
      enhanced: false,
      isCapHit: true,
      band: 'limitReached',
      usage: 50_000,
      limit: 50_000,
    });
    expect(
      mocks.useSkippableGraphQLQuery.mock.calls.map(([args]) => args.skip),
    ).toEqual([false, true]);
  });

  it.each([false, true])(
    'fetches nothing until the flag is ready (value %s)',
    (value) => {
      flag = { value, isReady: false };

      expect(useExecutionLimit()).toBeNull();
      expect(
        mocks.useSkippableGraphQLQuery.mock.calls.map(([args]) => args.skip),
      ).toEqual([true, true]);
    },
  );

  it('uses the enhanced cap only when the flag is ready and enabled', () => {
    flag = { value: true, isReady: true };

    expect(useExecutionLimit()).toMatchObject({
      accountID: 'enhanced-account',
      enhanced: true,
      isCapHit: false,
      band: 'limitReached',
      isVercel: true,
      marketplaceBillingURL: 'https://vercel.com/billing',
    });
    expect(
      mocks.useSkippableGraphQLQuery.mock.calls.map(([args]) => args.skip),
    ).toEqual([true, false]);
  });

  it('hides overage-eligible legacy accounts', () => {
    legacyAccount.entitlements.executions.overageAllowed = true;

    expect(useExecutionLimit()).toBeNull();
  });

  it('returns null while the selected query has no account data', () => {
    mocks.useSkippableGraphQLQuery.mockReturnValue({ data: undefined });

    expect(useExecutionLimit()).toBeNull();
  });

  it('returns null for an unlimited legacy account', () => {
    legacyAccount.entitlements.executions.limit = null;

    expect(useExecutionLimit()).toBeNull();
  });

  it('does not fall back to legacy limits when the enhanced cap is null', () => {
    flag = { value: true, isReady: true };
    enhancedAccount.executionCap = null;

    expect(useExecutionLimit()).toBeNull();
  });
});
