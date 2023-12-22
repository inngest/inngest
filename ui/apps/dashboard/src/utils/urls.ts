import type { Route } from 'next';

export const WEBSITE_PRICING_URL = 'https://www.inngest.com/pricing';
export const WEBSITE_CONTACT_URL = 'https://www.inngest.com/contact';

export const DOCS_URLS = {
  SERVE: 'https://www.inngest.com/docs/sdk/serve',
};

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
  deploys({ envSlug }: { envSlug: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/deploys`;
  },
  deploy({ deployID, envSlug }: { deployID: string; envSlug: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/deploys/${deployID}`;
  },
  events({ envSlug }: { envSlug: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/events`;
  },
  eventType({ envSlug }: { envSlug: string; eventName: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/events/${encodeURIComponent(eventName)}`;
  },
  functions({ envSlug }: { envSlug: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/functions`;
  },
  function({ envSlug, functionSlug }: { envSlug: string; functionSlug: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/functions/${encodeURIComponent(functionSlug)}`;
  },
  functionRuns({ envSlug, functionSlug }: { envSlug: string; functionSlug: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/functions/${encodeURIComponent(functionSlug)}/logs`;
  },
  keys({ envSlug }: { envSlug: string }): Route {
    // @ts-expect-error
    return `/env/${envSlug}/manage/keys`;
  },
};
