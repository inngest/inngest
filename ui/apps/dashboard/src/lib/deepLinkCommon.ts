export type DashboardDeepLinkSearchParams = {
  acct?: string;
  expires?: string;
  sig?: string;
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
