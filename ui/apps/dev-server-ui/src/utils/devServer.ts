export const isHosted = import.meta.env.VITE_HOSTED === 'true';
export const portStorageKey = 'inngest-dev-server-port';

export function validPort(value: string): boolean {
  return /^\d+$/.test(value) && Number(value) >= 1 && Number(value) <= 65535;
}

export function getAPIOrigin(): string {
  if (!isHosted) return import.meta.env.VITE_PUBLIC_API_BASE_URL || '';
  let port = '8288';
  try {
    const saved = window.localStorage.getItem(portStorageKey);
    if (saved && validPort(saved)) port = saved;
  } catch {}
  return `http://localhost:${port}`;
}

export function createDevServerURL(path: string): string {
  const origin = getAPIOrigin();
  return origin ? new URL(path, origin).toString() : path;
}

export type ConnectionStatus =
  | 'checking'
  | 'connected'
  | 'unreachable'
  | 'blocked'
  | 'unauthorized'
  | 'wrong-service'
  | 'api-unavailable';

export async function localPermissionDenied(): Promise<boolean> {
  if (typeof navigator === 'undefined' || !navigator.permissions) return false;
  for (const name of ['loopback-network', 'local-network-access']) {
    try {
      const result = await navigator.permissions.query({
        name: name as PermissionName,
      });
      if (result.state === 'denied') return true;
    } catch {}
  }
  return false;
}

export async function probeDevServer(
  signal: AbortSignal,
): Promise<ConnectionStatus> {
  try {
    const response = await fetch(createDevServerURL('/dev'), {
      signal,
      cache: 'no-store',
    });
    if (response.status === 401 || response.status === 403)
      return 'unauthorized';
    if (!response.ok) return 'wrong-service';
    const info: unknown = await response.json().catch(() => null);
    if (
      !info ||
      typeof info !== 'object' ||
      !('version' in info) ||
      typeof info.version !== 'string' ||
      !('startOpts' in info) ||
      !info.startOpts ||
      typeof info.startOpts !== 'object' ||
      !('features' in info) ||
      !info.features ||
      typeof info.features !== 'object'
    )
      return 'wrong-service';

    const api = await fetch(createDevServerURL('/v0/gql'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: '{ __typename }' }),
      signal,
      cache: 'no-store',
    });
    if (api.status === 401 || api.status === 403) return 'unauthorized';
    if (!api.ok) return 'api-unavailable';
    const body = await api.json().catch(() => null);
    return body?.data?.__typename === 'Query' ? 'connected' : 'api-unavailable';
  } catch {
    return (await localPermissionDenied()) ? 'blocked' : 'unreachable';
  }
}
