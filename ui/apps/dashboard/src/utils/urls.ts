import type { FileRouteTypes } from '@/routeTree.gen';

export const WEBSITE_PRICING_URL = 'https://www.inngest.com/pricing';
export const WEBSITE_CONTACT_URL = 'https://www.inngest.com/contact';
export const DISCORD_URL = 'https://www.inngest.com/discord';

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
  apps({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/apps` as FileRouteTypes['to'];
  },
  app({
    envSlug,
    externalAppID,
  }: {
    envSlug: string;
    externalAppID: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/apps/${encodeURIComponent(
      externalAppID,
    )}` as FileRouteTypes['to'];
  },
  appSyncs({
    envSlug,
    externalAppID,
  }: {
    envSlug: string;
    externalAppID: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/apps/${encodeURIComponent(
      externalAppID,
    )}/syncs` as FileRouteTypes['to'];
  },
  billing({
    ref,
    tab,
    highlight,
  }: {
    ref?: string;
    tab?: string;
    highlight?: string;
  } = {}): FileRouteTypes['to'] {
    let path = '/billing';
    if (tab) {
      path += `/${tab}`;
    }

    const query = new URLSearchParams();
    if (highlight) {
      query.set('highlight', highlight);
    }
    if (ref) {
      query.set('ref', ref);
    }
    if (query.toString()) {
      path += `?${query.toString()}`;
    }

    return path as FileRouteTypes['to'];
  },
  billingUsage({
    dimension,
    previous,
  }: {
    dimension?: string;
    previous?: boolean;
  } = {}): FileRouteTypes['to'] {
    let path = '/billing/usage';
    const query = new URLSearchParams();
    if (dimension) {
      query.set('dimension', dimension);
    }
    if (previous) {
      query.set('previous', previous.toString());
    }
    if (query.toString()) {
      path += `?${query.toString()}`;
    }
    return path as FileRouteTypes['to'];
  },
  createApp({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/apps/sync-new` as FileRouteTypes['to'];
  },
  eventPopout({
    envSlug,
    eventID,
  }: {
    envSlug: string;
    eventID: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/events/${eventID}` as FileRouteTypes['to'];
  },
  eventType({
    envSlug,
    eventName,
  }: {
    envSlug: string;
    eventName: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/event-types/${encodeURIComponent(
      eventName,
    )}` as FileRouteTypes['to'];
  },
  eventTypes({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/event-types` as FileRouteTypes['to'];
  },
  eventTypeEvents({
    envSlug,
    eventName,
  }: {
    envSlug: string;
    eventName: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/event-types/${encodeURIComponent(
      eventName,
    )}/events` as FileRouteTypes['to'];
  },
  envs(): FileRouteTypes['to'] {
    return '/env' as FileRouteTypes['to'];
  },
  functions({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/functions` as FileRouteTypes['to'];
  },
  function({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}` as FileRouteTypes['to'];
  },
  functionReplays({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/replays` as FileRouteTypes['to'];
  },
  functionReplay({
    envSlug,
    functionSlug,
    replayID,
  }: {
    envSlug: string;
    functionSlug: string;
    replayID: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/replays/${replayID}` as FileRouteTypes['to'];
  },
  functionCancellations({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/cancellations` as FileRouteTypes['to'];
  },
  insights({
    envSlug,
    ref,
  }: {
    envSlug: string;
    ref?: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/insights${
      ref ? `?ref=${ref}` : ''
    }` as FileRouteTypes['to'];
  },
  integrations(): FileRouteTypes['to'] {
    return `/settings/integrations` as FileRouteTypes['to'];
  },
  keys({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/manage/keys` as FileRouteTypes['to'];
  },
  pgIntegrationStep({
    integration,
    step,
  }: {
    integration: string;
    step?: string;
  }): FileRouteTypes['to'] {
    return `/settings/integrations/${integration}${
      step ? `/${step}` : ''
    }` as FileRouteTypes['to'];
  },
  onboarding({
    envSlug = 'production',
  }: { envSlug?: string } = {}): FileRouteTypes['to'] {
    return `/env/${envSlug}/onboarding` as FileRouteTypes['to'];
  },
  onboardingSteps({
    envSlug = 'production',
    step,
    ref,
  }: {
    envSlug?: string;
    step?: string;
    ref?: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/onboarding${step ? `/${step}` : ''}${
      ref ? `?ref=${ref}` : ''
    }` as FileRouteTypes['to'];
  },
  runPopout({
    envSlug,
    runID,
  }: {
    envSlug: string;
    runID: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/runs/${runID}` as FileRouteTypes['to'];
  },
  debugger({
    envSlug,
    functionSlug,
    runID,
  }: {
    envSlug: string;
    functionSlug: string;
    runID?: string;
  }): FileRouteTypes['to'] {
    return `/env/${envSlug}/debugger/${functionSlug}${
      runID ? `?runID=${runID}` : ''
    }` as FileRouteTypes['to'];
  },
  runs({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/runs` as FileRouteTypes['to'];
  },
  signingKeys({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/manage/signing-key` as FileRouteTypes['to'];
  },
  support({ ref }: { ref?: string } = {}): FileRouteTypes['to'] {
    return `https://support.inngest.com/${
      ref ? `?ref=${ref}` : ''
    }` as FileRouteTypes['to'];
  },
  unattachedSyncs({ envSlug }: { envSlug: string }): FileRouteTypes['to'] {
    return `/env/${envSlug}/unattached-syncs` as FileRouteTypes['to'];
  },
  vercel(): FileRouteTypes['to'] {
    return `/settings/integrations/vercel` as FileRouteTypes['to'];
  },
  vercelSetup(): FileRouteTypes['to'] {
    return `/settings/integrations/vercel/connect` as FileRouteTypes['to'];
  },
  neon(): FileRouteTypes['to'] {
    return `/settings/integrations/neon` as FileRouteTypes['to'];
  },
};
