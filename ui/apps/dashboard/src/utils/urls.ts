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
  app({ envSlug, externalAppID }: { envSlug: string; externalAppID: string }): Route {
    return `/env/${envSlug}/apps/${encodeURIComponent(externalAppID)}` as Route;
  },
  appSyncs({ envSlug, externalAppID }: { envSlug: string; externalAppID: string }): Route {
    return `/env/${envSlug}/apps/${encodeURIComponent(externalAppID)}/syncs` as Route;
  },
  billing({ ref, tab }: { ref?: string; tab?: string } = {}): Route {
    return `/billing${tab ? `/${tab}` : ''}${ref ? `?ref=${ref}` : ''}` as Route;
  },
  createApp({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/apps/sync-new` as Route;
  },
  event({
    envSlug,
    eventName,
    eventID,
  }: {
    envSlug: string;
    eventName: string;
    eventID: string;
  }): Route {
    return `/env/${envSlug}/events/${encodeURIComponent(eventName)}/logs/${eventID}` as Route;
  },
  events({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/events` as Route;
  },
  eventType({ envSlug, eventName }: { envSlug: string; eventName: string }): Route {
    return `/env/${envSlug}/events/${encodeURIComponent(eventName)}` as Route;
  },
  envs(): Route {
    return '/env' as Route;
  },
  functions({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/functions` as Route;
  },
  function({ envSlug, functionSlug }: { envSlug: string; functionSlug: string }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(functionSlug)}` as Route;
  },
  functionCancellations({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(functionSlug)}/cancellations` as Route;
  },
  keys({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/manage/keys` as Route;
  },
  pgIntegrationStep({ integration, step }: { integration: string; step?: string }): Route {
    return `/settings/integrations/${integration}${step ? `/${step}` : ''}` as Route;
  },
  onboarding({ envSlug = 'production' }: { envSlug?: string } = {}): Route {
    return `/env/${envSlug}/onboarding` as Route;
  },
  onboardingSteps({
    envSlug = 'production',
    step,
    ref,
  }: {
    envSlug?: string;
    step?: string;
    ref?: string;
  }): Route {
    return `/env/${envSlug}/onboarding${step ? `/${step}` : ''}${
      ref ? `?ref=${ref}` : ''
    }` as Route;
  },
  runPopout({ envSlug, runID }: { envSlug: string; runID: string }): Route {
    return `/env/${envSlug}/runs/${runID}` as Route;
  },
  runs({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/runs` as Route;
  },
  support({ ref }: { ref?: string } = {}): Route {
    return `/support${ref ? `?ref=${ref}` : ''}` as Route;
  },
  unattachedSyncs({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/unattached-syncs` as Route;
  },
  vercel(): Route {
    return `/settings/integrations/vercel` as Route;
  },
};
