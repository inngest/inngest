const fields = {
  first_utm_source: 'utmSource',
  first_utm_medium: 'utmMedium',
  first_utm_campaign: 'utmCampaign',
  first_utm_content: 'utmContent',
  first_utm_term: 'utmTerm',
  first_landing_url: 'firstLandingUrl',
} as const;

export const getSignupAttribution = (): Record<string, string> => {
  if (typeof document === 'undefined') return {};

  try {
    const prefix = 'inngest_first_touch=';
    const cookie = document.cookie
      .split(';')
      .map((value) => value.trim())
      .find((value) => value.startsWith(prefix));
    if (!cookie) return {};

    const firstTouch: unknown = JSON.parse(
      decodeURIComponent(cookie.slice(prefix.length)),
    );
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
