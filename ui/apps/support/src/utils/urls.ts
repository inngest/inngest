export const WEBSITE_PRICING_URL = "https://www.inngest.com/pricing";
export const WEBSITE_CONTACT_URL = "https://www.inngest.com/contact";
export const DISCORD_URL = "https://www.inngest.com/discord";

export const DOCS_URLS = {
  SERVE: "https://www.inngest.com/docs/sdk/serve",
};

// Dashboard URLs - uses VITE_APP_URL for the dashboard base URL
export function getDashboardBillingUrl(): string {
  const baseUrl = import.meta.env.VITE_APP_URL || "https://app.inngest.com";
  return `${baseUrl}/billing`;
}

export const pathCreator = {
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

  support({ ref }: { ref?: string } = {}): string {
    return `/support${ref ? `?ref=${ref}` : ""}`;
  },
};
