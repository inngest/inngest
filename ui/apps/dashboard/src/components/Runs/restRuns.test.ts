import { afterEach, describe, expect, it, vi } from 'vitest';

import type { InngestAPIFetch } from '@/queries/useInngestAPIFetch';

import {
  decodeRunsFrontier,
  fetchRunsPage,
  restFunctionRunToTableRun,
  restRunsRefetchInterval,
} from './restRuns';

const apiFetch: InngestAPIFetch = (pathname, init) =>
  fetch(new URL(pathname, 'https://api.example.com'), init);

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('restFunctionRunToTableRun', () => {
  it('maps complete and optional run fields', () => {
    expect(
      restFunctionRunToTableRun({
        id: 'run-1',
        function: { id: 'fn-id', name: 'Function', slug: 'function' },
        app: { id: 'app' },
        status: 'COMPLETED',
        queuedAt: '2026-08-31T10:00:00Z',
        startedAt: '2026-08-31T10:00:01Z',
        endedAt: '2026-08-31T10:00:03Z',
        durationMs: '2000',
        trigger: {
          eventName: 'app/tested',
          isBatch: true,
          cronSchedule: '*/5 * * * *',
        },
        isDeferred: true,
        hasAi: true,
      }),
    ).toMatchObject({
      id: 'run-1',
      app: { externalID: 'app', name: 'app' },
      function: { name: 'Function', slug: 'function' },
      durationMS: 2000,
      eventName: 'app/tested',
      isBatch: true,
      isDeferred: true,
      hasAI: true,
    });
  });

  it('maps sparse rows and rejects missing references', () => {
    const row = restFunctionRunToTableRun({
      id: 'run-1',
      function: { id: 'fn-id', name: 'Function' },
      app: { id: 'app' },
      status: 'RUNNING',
      queuedAt: '2026-08-31T10:00:00Z',
    });
    expect(row).toMatchObject({
      function: { slug: 'fn-id' },
      durationMS: null,
      eventName: null,
      isBatch: false,
      isDeferred: false,
    });
    expect(() =>
      restFunctionRunToTableRun({
        id: 'run-1',
        status: 'RUNNING',
        queuedAt: '2026-08-31T10:00:00Z',
      }),
    ).toThrow('missing required');
  });
});

describe('restRunsRefetchInterval', () => {
  it.each([
    { name: 'before the first page', hasCEL: false, pages: 0, want: false },
    { name: 'while only the first page is cached', hasCEL: false, pages: 1, want: 1000 },
    { name: 'after pagination begins', hasCEL: false, pages: 2, want: false },
    { name: 'during CEL search', hasCEL: true, pages: 1, want: false },
  ])('$name', ({ hasCEL, pages, want }) => {
    expect(restRunsRefetchInterval(hasCEL, pages)).toBe(want);
  });
});

it('decodes the selected cursor frontier', () => {
  const cursor = btoa(
    JSON.stringify({
      c: { started_at: { f: 'started_at', v: 1788170400000 } },
    }),
  );
  expect(decodeRunsFrontier(cursor, 'STARTED_AT')?.toISOString()).toBe(
    '2026-08-31T10:00:00.000Z',
  );
});

it('normalizes omitted protobuf defaults on empty terminal pages', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          metadata: { fetchedAt: '2026-08-31T10:00:00Z' },
          page: { cursor: 'frontier', limit: 40 },
        }),
        { status: 200 },
      ),
    ),
  );

  await expect(
    fetchRunsPage(apiFetch, '/v2/runs', new URLSearchParams()),
  ).resolves.toMatchObject({
    data: [],
    page: { cursor: 'frontier', hasMore: false, limit: 40 },
  });
});

it('backs off and retries rate-limited page requests', async () => {
  vi.useFakeTimers();
  const fetch = vi
    .fn()
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ errors: [{ message: 'slow down' }] }), {
        status: 429,
        headers: { 'Retry-After': '2' },
      }),
    )
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: [],
          page: { cursor: 'next', hasMore: false, limit: 40 },
        }),
        { status: 200 },
      ),
    );
  vi.stubGlobal('fetch', fetch);

  const result = fetchRunsPage(apiFetch, '/v2/runs', new URLSearchParams());
  await vi.advanceTimersByTimeAsync(1999);
  expect(fetch).toHaveBeenCalledTimes(1);
  await vi.advanceTimersByTimeAsync(1);
  await expect(result).resolves.toMatchObject({
    page: { cursor: 'next', hasMore: false },
  });
  expect(fetch).toHaveBeenCalledTimes(2);
});

it('stops retrying after three rate-limit retries', async () => {
  vi.useFakeTimers();
  const fetch = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ errors: [{ message: 'slow down' }] }), {
      status: 429,
      headers: { 'Retry-After': '1' },
    }),
  );
  vi.stubGlobal('fetch', fetch);

  const result = fetchRunsPage(apiFetch, '/v2/runs', new URLSearchParams());
  const rejection = expect(result).rejects.toMatchObject({ status: 429 });
  await vi.advanceTimersByTimeAsync(3000);
  await rejection;
  expect(fetch).toHaveBeenCalledTimes(4);
});

it('uses exponential backoff without a Retry-After header', async () => {
  vi.useFakeTimers();
  const rateLimited = () =>
    new Response(JSON.stringify({ errors: [{ message: 'slow down' }] }), {
      status: 429,
    });
  const fetch = vi
    .fn()
    .mockResolvedValueOnce(rateLimited())
    .mockResolvedValueOnce(rateLimited())
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: [],
          page: { cursor: 'next', hasMore: false, limit: 40 },
        }),
        { status: 200 },
      ),
    );
  vi.stubGlobal('fetch', fetch);

  const result = fetchRunsPage(apiFetch, '/v2/runs', new URLSearchParams());
  await vi.advanceTimersByTimeAsync(999);
  expect(fetch).toHaveBeenCalledTimes(1);
  await vi.advanceTimersByTimeAsync(1);
  expect(fetch).toHaveBeenCalledTimes(2);
  await vi.advanceTimersByTimeAsync(1999);
  expect(fetch).toHaveBeenCalledTimes(2);
  await vi.advanceTimersByTimeAsync(1);
  await expect(result).resolves.toMatchObject({ page: { cursor: 'next' } });
  expect(fetch).toHaveBeenCalledTimes(3);
});

it('cancels during rate-limit backoff', async () => {
  vi.useFakeTimers();
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ errors: [{ message: 'slow down' }] }), {
        status: 429,
        headers: { 'Retry-After': '60' },
      }),
    ),
  );
  const controller = new AbortController();

  const result = fetchRunsPage(
    apiFetch,
    '/v2/runs',
    new URLSearchParams(),
    controller.signal,
  );
  await vi.advanceTimersByTimeAsync(0);
  controller.abort();

  await expect(result).rejects.toMatchObject({ name: 'AbortError' });
});
