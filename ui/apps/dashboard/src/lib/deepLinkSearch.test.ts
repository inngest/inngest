import {
  sanitizeRedirectUrl,
  validateAgentDeepLinkSearch,
  validateDashboardDeepLinkSearch,
  validateSwitchOrganizationSearch,
} from '@/lib/deepLinkSearch';
import { describe, expect, it } from 'vitest';

describe('deepLinkSearch', () => {
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
        expires: 100,
        sig: 'a'.repeat(64),
      }),
    ).toEqual({
      acct: 'acct_123',
      expires: undefined,
      sig: 'a'.repeat(64),
    });
  });
});
