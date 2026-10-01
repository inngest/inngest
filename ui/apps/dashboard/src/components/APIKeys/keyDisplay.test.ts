import { describe, expect, it } from 'vitest';

import { EnvironmentType } from '@/gql/graphql';
import { apiKeyEnvironment, apiKeyStatus } from './keyDisplay';

describe('API key status', () => {
  const now = Date.parse('2026-09-23T12:00:00Z');

  it.each([
    {
      name: 'no expiration',
      expiresAt: null,
      revokedAt: null,
      expected: 'Active',
    },
    {
      name: 'future expiration',
      expiresAt: '2026-09-24T12:00:00Z',
      revokedAt: null,
      expected: 'Active',
    },
    {
      name: 'expired',
      expiresAt: '2026-09-22T12:00:00Z',
      revokedAt: null,
      expected: 'Expired',
    },
    {
      name: 'expires now',
      expiresAt: '2026-09-23T12:00:00Z',
      revokedAt: null,
      expected: 'Expired',
    },
    {
      name: 'revoked without expiration',
      expiresAt: null,
      revokedAt: '2026-09-22T12:00:00Z',
      expected: 'Revoked',
    },
    {
      name: 'revoked and expired',
      expiresAt: '2026-09-22T12:00:00Z',
      revokedAt: '2026-09-21T12:00:00Z',
      expected: 'Revoked',
    },
  ])('$name', ({ expected, ...key }) => {
    expect(apiKeyStatus(key, now)).toBe(expected);
  });
});

describe('API key environment', () => {
  it('labels account-wide access', () => {
    expect(apiKeyEnvironment({ env: null })).toBe('All environments');
  });

  it.each([
    {
      type: EnvironmentType.Production,
      name: 'production',
      expected: 'production',
    },
    {
      type: EnvironmentType.BranchChild,
      name: 'feature-checkout',
      expected: 'feature-checkout',
    },
    {
      type: EnvironmentType.BranchParent,
      name: 'branch',
      expected: 'Branch environments',
    },
  ])('$type', ({ type, name, expected }) => {
    expect(apiKeyEnvironment({ env: { id: 'env-id', type, name } })).toBe(
      expected,
    );
  });
});
