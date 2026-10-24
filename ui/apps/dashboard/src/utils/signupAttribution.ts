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
  last_utm_campaign: 'lastUtmCampaign',
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

const writeFirstTouch = (
  payload: Record<string, unknown>,
  maxAgeSeconds: number,
): void => {
  const attributes = [
    `${FIRST_TOUCH_COOKIE}=${encodeURIComponent(JSON.stringify(payload))}`,
    'path=/',
    `max-age=${maxAgeSeconds}`,
    'SameSite=Lax',
  ];
  const { hostname, protocol } = window.location;
  if (hostname === 'inngest.com' || hostname.endsWith('.inngest.com')) {
    attributes.push('domain=inngest.com');
  }
  if (protocol === 'https:') attributes.push('Secure');

  document.cookie = attributes.join('; ');
};

// the dashboard writes inngest_first_touch for campaign links that skip the
// website.  the cookie carries attribution to Clerk at signup.  later campaign
// URLs update last_utm_campaign and keep the original first fields.
export const ensureFirstTouch = (): void => {
  if (typeof document === 'undefined' || typeof window === 'undefined') return;

  try {
    const params = new URLSearchParams(window.location.search);
    const campaign = params.get('utm_campaign');
    const existing = readCookie(FIRST_TOUCH_COOKIE);
    if (existing) {
      const parsed: unknown = JSON.parse(decodeURIComponent(existing));
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed))
        return;

      const payload = parsed as Record<string, unknown>;
      const storedLast =
        typeof payload.last_utm_campaign === 'string'
          ? payload.last_utm_campaign
          : '';
      const storedFirst =
        typeof payload.first_utm_campaign === 'string'
          ? payload.first_utm_campaign
          : '';
      const lastCampaign = campaign || storedLast || storedFirst;
      if (!lastCampaign || storedLast === lastCampaign) {
        return;
      }

      const firstSeen = Date.parse(
        typeof payload.first_seen_at === 'string' ? payload.first_seen_at : '',
      );
      const maxAgeSeconds = Number.isNaN(firstSeen)
        ? SIX_MONTHS_SECONDS
        : Math.max(
            0,
            Math.floor(
              (firstSeen + SIX_MONTHS_SECONDS * 1000 - Date.now()) / 1000,
            ),
          );
      writeFirstTouch(
        { ...payload, last_utm_campaign: lastCampaign },
        maxAgeSeconds,
      );
      return;
    }

    const payload: Record<string, string> = {};
    for (const key of UTM_KEYS) {
      const value = params.get(key);
      if (value) payload[`first_${key}`] = value;
    }
    if (Object.keys(payload).length === 0) return;

    if (campaign) payload.last_utm_campaign = campaign;
    payload.first_landing_url = window.location.href;
    if (document.referrer) payload.first_referrer = document.referrer;
    payload.first_seen_at = new Date().toISOString();

    writeFirstTouch(payload, SIX_MONTHS_SECONDS);
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
