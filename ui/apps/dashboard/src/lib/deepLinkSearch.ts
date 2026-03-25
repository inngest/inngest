import type { DashboardDeepLinkSearchParams } from '@/lib/deepLinkCommon';

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
