export const WEBSITE_PRICING_URL = "https://www.inngest.com/pricing";
export const WEBSITE_CONTACT_URL = "https://www.inngest.com/contact";
export const DISCORD_URL = "https://www.inngest.com/discord";

export const DOCS_URLS = {
  SERVE: "https://www.inngest.com/docs/sdk/serve",
};

export const skipCacheSearchParam = {
  name: "skipCache",
  value: "true",
} as const;

/**
 * Adds a query param that asks data fetchers to skip their cache.
 */
export function setSkipCacheSearchParam(url: string): string {
  let value = `${skipCacheSearchParam.name}=${skipCacheSearchParam.value}`;
  if (url.includes("?")) {
    url += "&" + value;
  } else {
    url += "?" + value;
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
  apps({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/apps`;
  },
  app({
    envSlug,
    externalAppID,
  }: {
    envSlug: string;
    externalAppID: string;
  }): string {
    return `/env/${envSlug}/apps/${encodeURIComponent(externalAppID)}`;
  },
  appSyncs({
    envSlug,
    externalAppID,
  }: {
    envSlug: string;
    externalAppID: string;
  }): string {
    return `/env/${envSlug}/apps/${encodeURIComponent(externalAppID)}/syncs`;
  },
  billing({
    ref,
    tab,
    highlight,
  }: { ref?: string; tab?: string; highlight?: string } = {}): string {
    let path = "/billing";
    if (tab) {
      path += `/${tab}`;
    }

    const query = new URLSearchParams();
    if (highlight) {
      query.set("highlight", highlight);
    }
    if (ref) {
      query.set("ref", ref);
    }
    if (query.toString()) {
      path += `?${query.toString()}`;
    }

    return path;
  },
  createApp({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/apps/sync-new`;
  },
  eventPopout({
    envSlug,
    eventID,
  }: {
    envSlug: string;
    eventID: string;
  }): string {
    return `/env/${envSlug}/events/${eventID}`;
  },
  eventType({
    envSlug,
    eventName,
  }: {
    envSlug: string;
    eventName: string;
  }): string {
    return `/env/${envSlug}/event-types/${encodeURIComponent(eventName)}`;
  },
  eventTypes({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/event-types`;
  },
  eventTypeEvents({
    envSlug,
    eventName,
  }: {
    envSlug: string;
    eventName: string;
  }): string {
    return `/env/${envSlug}/event-types/${encodeURIComponent(
      eventName,
    )}/events`;
  },
  envs(): string {
    return "/env";
  },
  functions({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/functions`;
  },
  function({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): string {
    return `/env/${envSlug}/functions/${encodeURIComponent(functionSlug)}`;
  },
  functionReplays({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): string {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/replays`;
  },
  functionReplay({
    envSlug,
    functionSlug,
    replayID,
  }: {
    envSlug: string;
    functionSlug: string;
    replayID: string;
  }): string {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/replays/${replayID}`;
  },
  functionCancellations({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): string {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/cancellations`;
  },
  insights({ envSlug, ref }: { envSlug: string; ref?: string }): string {
    return `/env/${envSlug}/insights${ref ? `?ref=${ref}` : ""}`;
  },
  integrations(): string {
    return `/settings/integrations`;
  },
  keys({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/manage/keys`;
  },
  pgIntegrationStep({
    integration,
    step,
  }: {
    integration: string;
    step?: string;
  }): string {
    return `/settings/integrations/${integration}${step ? `/${step}` : ""}`;
  },
  onboarding({ envSlug = "production" }: { envSlug?: string } = {}): string {
    return `/env/${envSlug}/onboarding`;
  },
  onboardingSteps({
    envSlug = "production",
    step,
    ref,
  }: {
    envSlug?: string;
    step?: string;
    ref?: string;
  }): string {
    return `/env/${envSlug}/onboarding${step ? `/${step}` : ""}${
      ref ? `?ref=${ref}` : ""
    }`;
  },
  runPopout({ envSlug, runID }: { envSlug: string; runID: string }): string {
    return `/env/${envSlug}/runs/${runID}`;
  },
  debugger({
    envSlug,
    functionSlug,
    runID,
  }: {
    envSlug: string;
    functionSlug: string;
    runID?: string;
  }): string {
    return `/env/${envSlug}/debugger/${functionSlug}${
      runID ? `?runID=${runID}` : ""
    }`;
  },
  runs({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/runs`;
  },
  signingKeys({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/manage/signing-key`;
  },
  support({ ref }: { ref?: string } = {}): string {
    return `/support${ref ? `?ref=${ref}` : ""}`;
  },
  unattachedSyncs({ envSlug }: { envSlug: string }): string {
    return `/env/${envSlug}/unattached-syncs`;
  },
  vercel(): string {
    return `/settings/integrations/vercel`;
  },
  vercelSetup(): string {
    return `/settings/integrations/vercel/connect`;
  },
  neon(): string {
    return `/settings/integrations/neon`;
  },
};
