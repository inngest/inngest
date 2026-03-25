export type DashboardDeepLinkSearchParams = {
  acct?: string;
  expires?: string;
  sig?: string;
};

export type AgentDeepLinkSearch = {
  organization_id?: string;
  redirect_url?: string;
  ticket?: string;
};

export type SwitchOrganizationSearchParams = {
  organization_id?: string;
  redirect_url?: string;
};

export type RedirectUrlSearchParams = {
  redirect_url?: string;
};

export function hasDeepLinkParams(
  search: DashboardDeepLinkSearchParams,
): boolean {
  return Boolean(search.acct || search.expires || search.sig);
}

export function stripDeepLinkParams(href: string): string {
  const url = new URL(href, 'https://app.inngest.com');

  url.searchParams.delete('acct');
  url.searchParams.delete('expires');
  url.searchParams.delete('sig');

  return `${url.pathname}${url.search}${url.hash}`;
}

export function isValidDeepLink(
  search: DashboardDeepLinkSearchParams,
): search is Required<DashboardDeepLinkSearchParams> {
  return (
    typeof search.acct === 'string' &&
    search.acct.length > 0 &&
    typeof search.expires === 'string' &&
    /^\d+$/.test(search.expires) &&
    typeof search.sig === 'string' &&
    /^[a-f0-9]{64}$/i.test(search.sig)
  );
}

export function isExpired(
  expires: string,
  nowInSeconds = Math.floor(Date.now() / 1000),
): boolean {
  return Number(expires) < nowInSeconds;
}

export function sanitizeRedirectUrl(value: unknown): string | undefined {
  return typeof value === 'string' &&
    value.startsWith('/') &&
    !value.startsWith('//')
    ? value
    : undefined;
}

export function validateAgentDeepLinkSearch(
  search: Record<string, unknown>,
): AgentDeepLinkSearch {
  return {
    organization_id:
      typeof search.organization_id === 'string'
        ? search.organization_id
        : undefined,
    redirect_url: sanitizeRedirectUrl(search.redirect_url),
    ticket: typeof search.ticket === 'string' ? search.ticket : undefined,
  };
}

export function validateSwitchOrganizationSearch(
  search: Record<string, unknown>,
): SwitchOrganizationSearchParams {
  return {
    organization_id:
      typeof search.organization_id === 'string'
        ? search.organization_id
        : undefined,
    redirect_url: sanitizeRedirectUrl(search.redirect_url),
  };
}

export function validateDashboardDeepLinkSearch(
  search: Record<string, unknown>,
): DashboardDeepLinkSearchParams {
  return {
    acct: typeof search.acct === 'string' ? search.acct : undefined,
    expires: typeof search.expires === 'string' ? search.expires : undefined,
    sig: typeof search.sig === 'string' ? search.sig : undefined,
  };
}

export function validateRedirectUrlSearch(
  search: Record<string, unknown>,
): RedirectUrlSearchParams {
  return {
    redirect_url: sanitizeRedirectUrl(search.redirect_url),
  };
}
