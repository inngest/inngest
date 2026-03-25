import {
  hasDeepLinkParams,
  isExpired,
  isValidDeepLink,
  stripDeepLinkParams,
} from '@/lib/deepLinkCommon';
import { describe, expect, it } from 'vitest';

describe('deepLinkCommon', () => {
  it('treats the exact expiry second as still valid', () => {
    expect(isExpired('100', 100)).toBe(false);
    expect(isExpired('99', 100)).toBe(true);
  });

  it('detects when any deep-link parameter is present', () => {
    expect(hasDeepLinkParams({})).toBe(false);
    expect(hasDeepLinkParams({ acct: 'acct_123' })).toBe(true);
    expect(hasDeepLinkParams({ expires: '100' })).toBe(true);
    expect(hasDeepLinkParams({ sig: 'a'.repeat(64) })).toBe(true);
  });

  it('strips deep-link params while preserving the rest of the URL', () => {
    expect(
      stripDeepLinkParams(
        '/env/test?foo=bar&acct=acct_123&expires=100&sig=abc#section',
      ),
    ).toBe('/env/test?foo=bar#section');
  });

  it('only accepts a well-formed deep link payload', () => {
    expect(
      isValidDeepLink({
        acct: 'acct_123',
        expires: '100',
        sig: 'a'.repeat(64),
      }),
    ).toBe(true);

    expect(
      isValidDeepLink({
        acct: 'acct_123',
        expires: 'not-a-number',
        sig: 'a'.repeat(64),
      }),
    ).toBe(false);

    expect(
      isValidDeepLink({
        acct: 'acct_123',
        expires: '100',
        sig: 'invalid',
      }),
    ).toBe(false);
  });
});
