import { afterEach, expect, it, vi } from 'vitest';

import { createInngestAPIFetch } from './useInngestAPIFetch';

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

it('authenticates Clerk requests and preserves endpoint options', async () => {
  vi.stubEnv('VITE_API_URL', 'https://api.example.com');
  const getToken = vi.fn().mockResolvedValue('session-token');
  const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', fetch);
  const signal = new AbortController().signal;

  const apiFetch = createInngestAPIFetch(getToken, 'branch');
  await apiFetch('/v2/runs?limit=40', {
    headers: { Accept: 'application/json' },
    method: 'GET',
    signal,
  });

  expect(getToken).toHaveBeenCalledWith({ skipCache: true });
  const [url, init] = fetch.mock.calls[0] as [URL, RequestInit];
  expect(url.toString()).toBe('https://api.example.com/v2/runs?limit=40');
  expect(init).toMatchObject({
    credentials: 'include',
    method: 'GET',
    signal,
  });
  const headers = new Headers(init.headers);
  expect(headers.get('Accept')).toBe('application/json');
  expect(headers.get('Authorization')).toBe('Bearer session-token');
  expect(headers.get('X-Inngest-Env')).toBe('branch');
});

it('falls back to cookie authentication when Clerk has no token', async () => {
  vi.stubEnv('VITE_API_URL', 'https://api.example.com');
  const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', fetch);

  const apiFetch = createInngestAPIFetch(vi.fn().mockResolvedValue(null));
  await apiFetch('/v2/runs');

  const [, init] = fetch.mock.calls[0] as [URL, RequestInit];
  expect(init.credentials).toBe('include');
  const headers = new Headers(init.headers);
  expect(headers.has('Authorization')).toBe(false);
  expect(headers.has('X-Inngest-Env')).toBe(false);
});
