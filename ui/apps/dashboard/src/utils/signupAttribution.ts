const FIRST_TOUCH_COOKIE = 'inngest_first_touch';

const SIX_MONTHS_SECONDS = 6 * 30 * 24 * 60 * 60;

const UTM_KEYS = [
  'utm_source',
  'utm_medium',
  'utm_campaign',
  'utm_content',
  'utm_term',
] as const;

const fields = {
  first_utm_source: 'utmSource',
  first_utm_medium: 'utmMedium',
  first_utm_campaign: 'utmCampaign',
  first_utm_content: 'utmContent',
  first_utm_term: 'utmTerm',
  first_landing_url: 'firstLandingUrl',
} as const;

// Session cookie for the GA4 web stream G-4YPM75W7D9, the measurement ID used
// by the GA4 tag in GTM-5VCKJ23T on www.inngest.com and app.inngest.com.
const GA4_SESSION_COOKIE = '_ga_4YPM75W7D9';

// gclid/gbraid/wbraid are URL-safe tokens; anything else is not a click ID.
const CLICK_ID_PATTERN = /^[A-Za-z0-9_-]{1,512}$/;

const readCookie = (name: string): string | undefined => {
  if (typeof document === 'undefined') return undefined;

  const prefix = `${name}=`;
  return document.cookie
    .split(';')
    .map((value) => value.trim())
    .find((value) => value.startsWith(prefix))
    ?.slice(prefix.length);
};

/**
 * Writes the first-touch cookie from the current URL when none exists.
 *
 * The website writes the same cookie on www.inngest.com. Campaign links that
 * redirect straight to app.inngest.com skip the website, so without this the
 * campaign never reaches sign-up. Only runs when the URL carries utm
 * parameters, so a plain visit does not claim first touch ahead of a later
 * campaign visit. Never overwrites an existing cookie.
 */
export const ensureFirstTouch = (): void => {
  if (typeof document === 'undefined' || typeof window === 'undefined') return;

  try {
    if (readCookie(FIRST_TOUCH_COOKIE)) return;

    const params = new URLSearchParams(window.location.search);
    const payload: Record<string, string> = {};
    for (const key of UTM_KEYS) {
      const value = params.get(key);
      if (value) payload[`first_${key}`] = value;
    }
    if (Object.keys(payload).length === 0) return;

    payload.first_landing_url = window.location.href;
    if (document.referrer) payload.first_referrer = document.referrer;
    payload.first_seen_at = new Date().toISOString();

    const attributes = [
      `${FIRST_TOUCH_COOKIE}=${encodeURIComponent(JSON.stringify(payload))}`,
      'path=/',
      `max-age=${SIX_MONTHS_SECONDS}`,
      'SameSite=Lax',
    ];
    const { hostname, protocol } = window.location;
    if (hostname === 'inngest.com' || hostname.endsWith('.inngest.com')) {
      attributes.push('domain=inngest.com');
    }
    if (protocol === 'https:') attributes.push('Secure');

    document.cookie = attributes.join('; ');
  } catch {
    // best effort
  }
};

const readUrlParam = (name: string): string | undefined => {
  if (typeof window === 'undefined') return undefined;

  try {
    return new URLSearchParams(window.location.search).get(name) ?? undefined;
  } catch {
    return undefined;
  }
};

export const getSignupAttribution = (): Record<string, string> => {
  if (typeof document === 'undefined') return {};

  try {
    const cookie = readCookie(FIRST_TOUCH_COOKIE);
    if (!cookie) return {};

    const firstTouch: unknown = JSON.parse(decodeURIComponent(cookie));
    if (
      !firstTouch ||
      typeof firstTouch !== 'object' ||
      Array.isArray(firstTouch)
    ) {
      return {};
    }

    return Object.fromEntries(
      Object.entries(fields).flatMap(([cookieKey, metadataKey]) => {
        const value = (firstTouch as Record<string, unknown>)[cookieKey];
        return typeof value === 'string' && value ? [[metadataKey, value]] : [];
      }),
    );
  } catch {
    return {};
  }
};

/** `_ga` is `GA1.<n>.<random>.<timestamp>`; the client ID is the last two parts. */
export const parseGaClientId = (
  value: string | undefined,
): string | undefined => value?.match(/^GA\d+\.\d+\.(\d+\.\d+)$/)?.[1];

/**
 * `_ga_<stream>` holds the current session ID in one of two formats:
 * `GS1.1.<session>.<count>...` (legacy) or `GS2.1.s<session>$o<count>$...`
 * (Google switched to this in 2025).
 */
export const parseGaSessionId = (
  value: string | undefined,
): string | undefined => {
  if (!value) return undefined;

  if (value.startsWith('GS2.')) {
    const segments = value.split('.').slice(2).join('.').split('$');
    return segments.map((s) => s.match(/^s(\d+)$/)?.[1]).find(Boolean);
  }
  if (value.startsWith('GS1.')) {
    return value.match(/^GS1\.\d+\.(\d+)\./)?.[1];
  }
  return undefined;
};

/**
 * Conversion Linker cookies (`_gcl_aw`, `_gcl_gb`) are `GCL.<unix seconds>.<click ID>`.
 * The timestamp is dropped (not the click ID) when it can't be a valid Date, so
 * a malformed cookie can never make `toISOString()` throw and break the auth forms.
 */
export const parseGclCookie = (
  value: string | undefined,
): { clickId: string; clickedAtMs?: number } | undefined => {
  const match = value?.match(/^GCL\.(\d+)\.(.+)$/);
  if (!match || !CLICK_ID_PATTERN.test(match[2])) return undefined;
  const clickedAtMs = Number(match[1]) * 1000;
  return {
    clickId: match[2],
    clickedAtMs: Number.isNaN(new Date(clickedAtMs).getTime())
      ? undefined
      : clickedAtMs,
  };
};

/**
 * GA4 identifiers (so the server-side Account Created event can join the
 * visitor's GA4 session) and Google Ads click IDs (for conversion uploads).
 */
export const getGoogleAttribution = (): Record<string, string> => {
  const attribution: Record<string, string> = {};

  const gaClientId = parseGaClientId(readCookie('_ga'));
  if (gaClientId) attribution.gaClientId = gaClientId;

  const gaSessionId = parseGaSessionId(readCookie(GA4_SESSION_COOKIE));
  if (gaSessionId) attribution.gaSessionId = gaSessionId;

  // A click ID on the current URL is the newest click and may not have reached
  // the Conversion Linker yet. Otherwise use what the linker stored for
  // inngest.com (last click, 90 days).
  const urlClickIds = Object.fromEntries(
    (['gclid', 'gbraid', 'wbraid'] as const).flatMap((key) => {
      const value = readUrlParam(key);
      return value && CLICK_ID_PATTERN.test(value) ? [[key, value]] : [];
    }),
  );

  let clickedAtMs: number | undefined;
  if (Object.keys(urlClickIds).length > 0) {
    Object.assign(attribution, urlClickIds);
    clickedAtMs = Date.now();
  } else {
    const gclid = parseGclCookie(readCookie('_gcl_aw'));
    const gbraid = parseGclCookie(readCookie('_gcl_gb'));
    if (gclid) attribution.gclid = gclid.clickId;
    if (gbraid) attribution.gbraid = gbraid.clickId;
    const times = [gclid?.clickedAtMs, gbraid?.clickedAtMs].filter(
      (t): t is number => typeof t === 'number' && t > 0,
    );
    if (times.length > 0) clickedAtMs = Math.max(...times);
  }
  if (clickedAtMs && !Number.isNaN(new Date(clickedAtMs).getTime())) {
    attribution.clickTs = new Date(clickedAtMs).toISOString();
  }

  return attribution;
};

export const getSignupMetadata = (): Record<string, string> => {
  if (typeof document === 'undefined') return {};

  ensureFirstTouch();

  const anonymousID = readCookie('ajs_anonymous_id');

  return {
    ...(anonymousID && { anonymousID }),
    ...getSignupAttribution(),
    ...getGoogleAttribution(),
  };
};
