import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  getGoogleAttribution,
  getSignupAttribution,
  getSignupMetadata,
  parseGaClientId,
  parseGaSessionId,
  parseGclCookie,
} from './signupAttribution';

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

describe('parseGaClientId', () => {
  it.each([
    ['GA1.1.1247616174.1788106374', '1247616174.1788106374'],
    ['GA1.2.1247616174.1788106374', '1247616174.1788106374'],
    [undefined, undefined],
    ['', undefined],
    ['GA1.1.abc.123', undefined],
    ['1247616174.1788106374', undefined],
  ])('%s -> %s', (cookie, expected) => {
    expect(parseGaClientId(cookie)).toBe(expected);
  });
});

describe('parseGaSessionId', () => {
  it.each([
    ['GS2.1.s1790347813$o3$g0$t1790347813$j60$l0$h0', '1790347813'],
    ['GS2.1.o3$s1790347813$g0', '1790347813'],
    ['GS1.1.1675243172.3.1.1675243500.0.0.0', '1675243172'],
    [undefined, undefined],
    ['GS2.1.o3$g0', undefined],
    ['GS3.1.s123', undefined],
    ['garbage', undefined],
  ])('%s -> %s', (cookie, expected) => {
    expect(parseGaSessionId(cookie)).toBe(expected);
  });
});

describe('parseGclCookie', () => {
  it('reads the click ID and click time', () => {
    expect(parseGclCookie('GCL.1790347814.Cj0KCQ_abc-123')).toEqual({
      clickId: 'Cj0KCQ_abc-123',
      clickedAtMs: 1790347814000,
    });
  });

  it.each([undefined, '', 'GCL.abc.xyz', 'GCL.1790347814.', 'GCL.1.bad id'])(
    'ignores %s',
    (cookie) => {
      expect(parseGclCookie(cookie)).toBeUndefined();
    },
  );
});

describe('getGoogleAttribution', () => {
  const setLocation = (search: string) =>
    vi.stubGlobal('window', { location: { search } });

  it('returns nothing without cookies or a browser', () => {
    expect(getGoogleAttribution()).toEqual({});
  });

  it('reads GA IDs and Conversion Linker click IDs from cookies', () => {
    setCookie(
      [
        '_ga=GA1.1.1247616174.1788106374',
        '_ga_4YPM75W7D9=GS2.1.s1790347813$o3$g0$t1790347813$j60$l0$h0',
        '_ga_OTHER=GS2.1.s999$o1',
        '_gcl_aw=GCL.1790347814.gclid_from_cookie',
        '_gcl_gb=GCL.1790340000.gbraid_from_cookie',
      ].join('; '),
    );
    expect(getGoogleAttribution()).toEqual({
      gaClientId: '1247616174.1788106374',
      gaSessionId: '1790347813',
      gclid: 'gclid_from_cookie',
      gbraid: 'gbraid_from_cookie',
      clickTs: new Date(1790347814000).toISOString(),
    });
  });

  it('prefers click IDs on the current URL over stored ones', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-09-25T12:00:00Z'));
    setCookie('_gcl_aw=GCL.1790347814.older_click');
    setLocation('?utm_source=google&gclid=newer_click&wbraid=bad%20value');
    expect(getGoogleAttribution()).toEqual({
      gclid: 'newer_click',
      clickTs: '2026-09-25T12:00:00.000Z',
    });
    vi.useRealTimers();
  });

  it('is included in the Clerk signup metadata', () => {
    setCookie(
      'ajs_anonymous_id=anon; _ga=GA1.1.1.2; _gcl_aw=GCL.1790347814.abc',
    );
    expect(getSignupMetadata()).toEqual({
      anonymousID: 'anon',
      gaClientId: '1.2',
      gclid: 'abc',
      clickTs: new Date(1790347814000).toISOString(),
    });
  });
});
