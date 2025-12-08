import type { Route } from "next";

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
  apps({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/apps` as Route;
  },
  app({
    envSlug,
    externalAppID,
  }: {
    envSlug: string;
    externalAppID: string;
  }): Route {
    return `/env/${envSlug}/apps/${encodeURIComponent(externalAppID)}` as Route;
  },
  appSyncs({
    envSlug,
    externalAppID,
  }: {
    envSlug: string;
    externalAppID: string;
  }): Route {
    return `/env/${envSlug}/apps/${encodeURIComponent(
      externalAppID,
    )}/syncs` as Route;
  },
  billing({
    ref,
    tab,
    highlight,
  }: { ref?: string; tab?: string; highlight?: string } = {}): Route {
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

    return path as Route;
  },
  createApp({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/apps/sync-new` as Route;
  },
  eventPopout({
    envSlug,
    eventID,
  }: {
    envSlug: string;
    eventID: string;
  }): Route {
    return `/env/${envSlug}/events/${eventID}` as Route;
  },
  eventType({
    envSlug,
    eventName,
  }: {
    envSlug: string;
    eventName: string;
  }): Route {
    return `/env/${envSlug}/event-types/${encodeURIComponent(
      eventName,
    )}` as Route;
  },
  eventTypes({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/event-types` as Route;
  },
  eventTypeEvents({
    envSlug,
    eventName,
  }: {
    envSlug: string;
    eventName: string;
  }): Route {
    return `/env/${envSlug}/event-types/${encodeURIComponent(
      eventName,
    )}/events` as Route;
  },
  envs(): Route {
    return "/env" as Route;
  },
  functions({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/functions` as Route;
  },
  function({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}` as Route;
  },
  functionReplays({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/replays` as Route;
  },
  functionReplay({
    envSlug,
    functionSlug,
    replayID,
  }: {
    envSlug: string;
    functionSlug: string;
    replayID: string;
  }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/replays/${replayID}` as Route;
  },
  functionCancellations({
    envSlug,
    functionSlug,
  }: {
    envSlug: string;
    functionSlug: string;
  }): Route {
    return `/env/${envSlug}/functions/${encodeURIComponent(
      functionSlug,
    )}/cancellations` as Route;
  },
  insights({ envSlug, ref }: { envSlug: string; ref?: string }): Route {
    return `/env/${envSlug}/insights${ref ? `?ref=${ref}` : ""}` as Route;
  },
  integrations(): Route {
    return `/settings/integrations` as Route;
  },
  keys({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/manage/keys` as Route;
  },
  pgIntegrationStep({
    integration,
    step,
  }: {
    integration: string;
    step?: string;
  }): Route {
    return `/settings/integrations/${integration}${
      step ? `/${step}` : ""
    }` as Route;
  },
  onboarding({ envSlug = "production" }: { envSlug?: string } = {}): Route {
    return `/env/${envSlug}/onboarding` as Route;
  },
  onboardingSteps({
    envSlug = "production",
    step,
    ref,
  }: {
    envSlug?: string;
    step?: string;
    ref?: string;
  }): Route {
    return `/env/${envSlug}/onboarding${step ? `/${step}` : ""}${
      ref ? `?ref=${ref}` : ""
    }` as Route;
  },
  runPopout({ envSlug, runID }: { envSlug: string; runID: string }): Route {
    return `/env/${envSlug}/runs/${runID}` as Route;
  },
  debugger({
    envSlug,
    functionSlug,
    runID,
  }: {
    envSlug: string;
    functionSlug: string;
    runID?: string;
  }): Route {
    return `/env/${envSlug}/debugger/${functionSlug}${
      runID ? `?runID=${runID}` : ""
    }` as Route;
  },
  runs({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/runs` as Route;
  },
  signingKeys({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/manage/signing-key` as Route;
  },
  support({ ref }: { ref?: string } = {}): Route {
    return `/support${ref ? `?ref=${ref}` : ""}` as Route;
  },
  unattachedSyncs({ envSlug }: { envSlug: string }): Route {
    return `/env/${envSlug}/unattached-syncs` as Route;
  },
  vercel(): Route {
    return `/settings/integrations/vercel` as Route;
  },
  vercelSetup(): Route {
    return `/settings/integrations/vercel/connect` as Route;
  },
  neon(): Route {
    return `/settings/integrations/neon` as Route;
  },
};
