export type SetupClient = 'claude' | 'codex' | 'other';

export function safeRunsReturnTo(value: unknown): string | undefined {
  if (
    typeof value !== 'string' ||
    !value.startsWith('/') ||
    value.includes('\\') ||
    [...value].some((character) => character.charCodeAt(0) <= 32)
  )
    return;
  const url = new URL(value, 'https://app.inngest.com');
  if (url.origin !== 'https://app.inngest.com') return;
  if (!/^\/env\/[^/]+\/(?:runs|functions\/[^/]+\/runs)\/?$/.test(url.pathname))
    return;
  return `${url.pathname}${url.search}`;
}

export function validateRunAISetupSearch(search: Record<string, unknown>) {
  return {
    client: (['claude', 'codex', 'other'].includes(String(search.client))
      ? search.client
      : undefined) as SetupClient | undefined,
    returnTo: safeRunsReturnTo(search.returnTo),
  };
}

export function runAISetupHref(client: SetupClient, returnTo: string) {
  const params = new URLSearchParams({ client });
  const safeReturnTo = safeRunsReturnTo(returnTo);
  if (safeReturnTo) params.set('returnTo', safeReturnTo);
  return `/mcp/setup?${params.toString()}`;
}
