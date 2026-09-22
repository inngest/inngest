import { afterEach, describe, expect, it, vi } from 'vitest';

import { getSignupAttribution } from './signupAttribution';

afterEach(() => vi.unstubAllGlobals());

const setCookie = (cookie: string) => vi.stubGlobal('document', { cookie });

describe('getSignupAttribution', () => {
  it('returns no attribution during server rendering', () => {
    expect(getSignupAttribution()).toEqual({});
  });

  it('maps only the requested first-touch fields from an encoded cookie', () => {
    const firstTouch = {
      first_utm_source: 'newsletter',
      first_utm_medium: 'email',
      first_utm_campaign: 'launch & learn',
      first_utm_content: 'hero',
      first_utm_term: 'durable workflows',
      first_landing_url: 'https://www.inngest.com/?utm_source=newsletter&x=a=b',
      first_ref: 'referral',
      first_seen_at: '2026-09-16',
      anonymousID: 'must-not-overwrite',
    };
    setCookie(
      `other=value; inngest_first_touch=${encodeURIComponent(
        JSON.stringify(firstTouch),
      )}; ajs_anonymous_id=anon`,
    );
    expect(getSignupAttribution()).toEqual({
      utmSource: 'newsletter',
      utmMedium: 'email',
      utmCampaign: 'launch & learn',
      utmContent: 'hero',
      utmTerm: 'durable workflows',
      firstLandingUrl: firstTouch.first_landing_url,
    });
  });

  it.each([
    '',
    'other=value',
    'inngest_first_touch=',
    'inngest_first_touch=%E0%A4%A',
    'inngest_first_touch=invalid-json',
    'inngest_first_touch=null',
    'inngest_first_touch=[]',
    'inngest_first_touch=42',
    'inngest_first_touch="text"',
    'inngest_first_touch={}',
  ])('ignores absent or malformed cookies: %s', (cookie) => {
    setCookie(cookie);
    expect(getSignupAttribution()).toEqual({});
  });

  it('omits invalid fields while preserving valid strings and embedded equals signs', () => {
    setCookie(
      `inngest_first_touch=${JSON.stringify({
        first_utm_source: 'test',
        first_utm_medium: '',
        first_utm_campaign: 123,
        first_utm_content: {},
        first_utm_term: null,
        first_landing_url: 'https://www.inngest.com/?x=a=b',
      })}`,
    );
    expect(getSignupAttribution()).toEqual({
      utmSource: 'test',
      firstLandingUrl: 'https://www.inngest.com/?x=a=b',
    });
  });
});
