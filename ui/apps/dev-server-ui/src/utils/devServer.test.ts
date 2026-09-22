import { afterEach, describe, expect, it, vi } from 'vitest';

async function load(hosted = true) {
  vi.resetModules();
  vi.stubEnv('VITE_HOSTED', hosted ? 'true' : 'false');
  vi.stubEnv('VITE_PUBLIC_API_BASE_URL', '');
  return import('./devServer');
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

const info = { version: '1.0.0', startOpts: {}, features: {} };
const signal = () => new AbortController().signal;

describe('hosted localhost connection', () => {
  it('uses localhost without browser globals during prerender', async () => {
    const api = await load();
    expect(api.createDevServerURL('/v0/gql')).toBe(
      'http://localhost:8288/v0/gql',
    );
  });

  it('retains same-origin API paths for the embedded CLI build', async () => {
    const api = await load(false);
    expect(api.createDevServerURL('/dev')).toBe('/dev');
    vi.stubEnv('VITE_PUBLIC_API_BASE_URL', 'http://localhost:9999');
    expect(api.createDevServerURL('/dev')).toBe('http://localhost:9999/dev');
  });

  it('accepts only a local port override', async () => {
    const api = await load();
    const getItem = vi.fn().mockReturnValue('9999');
    vi.stubGlobal('window', { localStorage: { getItem } });
    expect(api.createDevServerURL('/dev')).toBe('http://localhost:9999/dev');
    for (const value of [
      '0',
      '65536',
      '-1',
      '1.5',
      'https://example.com',
      '8288/path',
    ]) {
      getItem.mockReturnValue(value);
      expect(api.getAPIOrigin()).toBe('http://localhost:8288');
    }
    expect(api.validPort('65535')).toBe(true);
  });

  it('checks both server identity and GraphQL without cached responses', async () => {
    const api = await load();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(Response.json(info))
      .mockResolvedValueOnce(Response.json({ data: { __typename: 'Query' } }));
    vi.stubGlobal('fetch', fetchMock);
    expect(await api.probeDevServer(signal())).toBe('connected');
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      'http://localhost:8288/dev',
      'http://localhost:8288/v0/gql',
    ]);
    for (const [, init] of fetchMock.mock.calls) {
      expect(init).toMatchObject({
        cache: 'no-store',
      });
    }
  });

  it('rejects an unrelated service on the selected port', async () => {
    const api = await load();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(Response.json({ version: 'nginx' })),
    );
    expect(await api.probeDevServer(signal())).toBe('wrong-service');
  });

  it('distinguishes authenticated servers and disabled UI APIs', async () => {
    const api = await load();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('', { status: 401 }));
    vi.stubGlobal('fetch', fetchMock);
    expect(await api.probeDevServer(signal())).toBe('unauthorized');
    fetchMock
      .mockResolvedValueOnce(Response.json(info))
      .mockResolvedValueOnce(new Response('', { status: 404 }));
    expect(await api.probeDevServer(signal())).toBe('api-unavailable');
  });

  it('handles offline and permission-denied failures without assuming the server is stopped', async () => {
    const api = await load();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new TypeError('Failed to fetch')),
    );
    vi.stubGlobal('navigator', {
      permissions: { query: vi.fn().mockRejectedValue(new TypeError()) },
    });
    expect(await api.probeDevServer(signal())).toBe('unreachable');
    vi.stubGlobal('navigator', {
      permissions: { query: vi.fn().mockResolvedValue({ state: 'denied' }) },
    });
    expect(await api.probeDevServer(signal())).toBe('blocked');
  });

  it('can reconnect after the local process restarts', async () => {
    const api = await load();
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockRejectedValueOnce(new TypeError())
        .mockResolvedValueOnce(Response.json(info))
        .mockResolvedValueOnce(
          Response.json({ data: { __typename: 'Query' } }),
        ),
    );
    expect(await api.probeDevServer(signal())).toBe('unreachable');
    expect(await api.probeDevServer(signal())).toBe('connected');
  });
});
