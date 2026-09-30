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

const readCookie = (name: string): string | undefined => {
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
 * campaign never reaches sign-up. Only runs when the URL carries utm or ref
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
    const ref = params.get('ref');
    if (ref) payload.first_ref = ref;
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

export const getSignupMetadata = (): Record<string, string> => {
  if (typeof document === 'undefined') return {};

  ensureFirstTouch();

  const anonymousID = readCookie('ajs_anonymous_id');

  return {
    ...(anonymousID && { anonymousID }),
    ...getSignupAttribution(),
  };
};
