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

  expect(getToken).toHaveBeenCalledOnce();
  expect(getToken).toHaveBeenCalledWith();
  expect(fetch).toHaveBeenCalledOnce();
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

it('refreshes the Clerk token and retries a 401 once', async () => {
  vi.stubEnv('VITE_API_URL', 'https://api.example.com');
  const getToken = vi
    .fn()
    .mockResolvedValueOnce('cached-token')
    .mockResolvedValueOnce('fresh-token');
  const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 401 }));
  vi.stubGlobal('fetch', fetch);

  const apiFetch = createInngestAPIFetch(getToken);
  const response = await apiFetch('/v2/runs');

  expect(response.status).toBe(401);
  expect(getToken.mock.calls).toEqual([[], [{ skipCache: true }]]);
  expect(fetch).toHaveBeenCalledTimes(2);
  const firstHeaders = new Headers(fetch.mock.calls[0]?.[1]?.headers);
  const retryHeaders = new Headers(fetch.mock.calls[1]?.[1]?.headers);
  expect(firstHeaders.get('Authorization')).toBe('Bearer cached-token');
  expect(retryHeaders.get('Authorization')).toBe('Bearer fresh-token');
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
