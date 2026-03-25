import {
  hasDeepLinkParams,
  isExpired,
  isValidDeepLink,
  sanitizeRedirectUrl,
  stripDeepLinkParams,
  validateAgentDeepLinkSearch,
  validateDashboardDeepLinkSearch,
  validateSwitchOrganizationSearch,
} from '@/lib/deepLinkUtils';
import { describe, expect, it } from 'vitest';

describe('deepLinkUtils', () => {
  it('treats the exact expiry second as still valid', () => {
    expect(isExpired('100', 100)).toBe(false);
    expect(isExpired('99', 100)).toBe(true);
  });

  it('detects when any deep-link parameter is present', () => {
    expect(hasDeepLinkParams({})).toBe(false);
    expect(hasDeepLinkParams({ acct: 'acct_123' })).toBe(true);
    expect(hasDeepLinkParams({ org: 'org_123' })).toBe(true);
    expect(hasDeepLinkParams({ expires: '100' })).toBe(true);
    expect(hasDeepLinkParams({ sig: 'a'.repeat(64) })).toBe(true);
  });

  it('strips deep-link params while preserving the rest of the URL', () => {
    expect(
      stripDeepLinkParams(
        '/env/test?foo=bar&acct=acct_123&org=org_123&expires=100&sig=abc#section',
      ),
    ).toBe('/env/test?foo=bar#section');
  });

  it('only accepts a well-formed deep link payload', () => {
    expect(
      isValidDeepLink({
        acct: 'acct_123',
        org: 'org_123',
        expires: '100',
        sig: 'a'.repeat(64),
      }),
    ).toBe(true);

    expect(
      isValidDeepLink({
        acct: 'acct_123',
        expires: '100',
        sig: 'a'.repeat(64),
      }),
    ).toBe(false);

    expect(
      isValidDeepLink({
        acct: 'acct_123',
        org: 'org_123',
        expires: 'not-a-number',
        sig: 'a'.repeat(64),
      }),
    ).toBe(false);

    expect(
      isValidDeepLink({
        acct: 'acct_123',
        org: 'org_123',
        expires: '100',
        sig: 'invalid',
      }),
    ).toBe(false);
  });

  it('only allows internal redirect URLs', () => {
    expect(sanitizeRedirectUrl('/env/test?foo=bar')).toBe('/env/test?foo=bar');
    expect(sanitizeRedirectUrl('https://evil.example')).toBeUndefined();
    expect(sanitizeRedirectUrl('//evil.example')).toBeUndefined();
    expect(sanitizeRedirectUrl('javascript:alert(1)')).toBeUndefined();
  });

  it('sanitizes agent deep-link search params', () => {
    expect(
      validateAgentDeepLinkSearch({
        organization_id: 'org_123',
        redirect_url: 'https://evil.example',
        ticket: 'ticket_123',
      }),
    ).toEqual({
      organization_id: 'org_123',
      redirect_url: undefined,
      ticket: 'ticket_123',
    });
  });

  it('sanitizes switch-organization search params', () => {
    expect(
      validateSwitchOrganizationSearch({
        organization_id: 'org_123',
        redirect_url: '/env/test',
      }),
    ).toEqual({
      organization_id: 'org_123',
      redirect_url: '/env/test',
    });
  });

  it('only keeps string dashboard deep-link params', () => {
    expect(
      validateDashboardDeepLinkSearch({
        acct: 'acct_123',
        org: 'org_123',
        expires: 100,
        sig: 'a'.repeat(64),
      }),
    ).toEqual({
      acct: 'acct_123',
      org: 'org_123',
      expires: undefined,
      sig: 'a'.repeat(64),
    });
  });
});
