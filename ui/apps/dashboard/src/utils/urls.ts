import type { Route } from 'next';

export const WEBSITE_PRICING_URL = 'https://www.inngest.com/pricing';
export const WEBSITE_CONTACT_URL = 'https://www.inngest.com/contact';

export const DOCS_URLS = {
  SERVE: 'https://www.inngest.com/docs/sdk/serve',
};

export const skipCacheSearchParam = {
  name: 'skipCache',
  value: 'true',
} as const;

/**
 * Adds a query param that asks data fetchers to skip their cache.
 */
export function setSkipCacheSearchParam(url: string): string {
  let value = `${skipCacheSearchParam.name}=${skipCacheSearchParam.value}`;
  if (url.includes('?')) {
    url += '&' + value;
  } else {
    url += '?' + value;
  }
  return url;
}

export function getManageKey(pathname: string) {
  const regex = /\/manage\/(\w+)/;
  const match = pathname.match(regex);
  if (match && match[1]) {
    return match[1];
  } else {
    return null;
  }
}

export const pathCreator = {
  apps({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/apps` as Route;
  },
  deploys({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/deploys` as Route;
  },
  deploy({ deployID, envSlug }: { deployID: string; envSlug: string }): Route {
    return `/env/${envSlug}/deploys/${deployID}` as Route;
  },
  events({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/events` as Route;
  },
  eventType({ envSlug, eventName }: { envSlug: string; eventName: string }): Route {
    return `/env/${envSlug}/events/${encodeURIComponent(eventName)}` as Route;
  },
  functions({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/functions` as Route;
  },
  function({ envSlug, functionSlug }: { envSlug: string; functionSlug: string }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(functionSlug)}` as Route;
  },
  functionRuns({ envSlug, functionSlug }: { envSlug: string; functionSlug: string }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(functionSlug)}/logs` as Route;
  },
  keys({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/manage/keys` as Route;
  },
};
