import { afterEach, describe, expect, it, vi } from 'vitest';

import type { InngestAPIFetch } from '@/queries/useInngestAPIFetch';

import {
  decodeRunsFrontier,
  fetchRunsPage,
  getRestAppIDs,
  restFunctionRunToTableRun,
} from './restRuns';
import { fetchRestRuns } from './useRunsPagination';

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
        deferredFrom: {
          functionSlug: 'parent-function',
          functionName: 'Parent function',
        },
      }),
    ).toMatchObject({
      id: 'run-1',
      app: { externalID: 'app', name: 'app' },
      function: { name: 'Function', slug: 'function' },
      durationMS: 2000,
      eventName: 'app/tested',
      isBatch: true,
      isDeferred: true,
      deferredFrom: [
        {
          function: { name: 'Parent function', slug: 'parent-function' },
        },
      ],
    });
  });

  it.each([
    {
      case: 'complete function metadata',
      function: { id: 'fn-id', name: 'Function', slug: 'function' },
      expected: { name: 'Function', slug: 'function' },
    },
    {
      case: 'a missing name with a slug',
      function: { id: 'fn-id', slug: 'function' },
      expected: { name: 'function', slug: 'function' },
    },
    {
      case: 'a missing name and slug with an ID',
      function: { id: 'fn-id' },
      expected: { name: 'fn-id', slug: 'fn-id' },
    },
    {
      case: 'an empty function reference',
      function: {},
      expected: { name: 'Function unavailable', slug: '' },
    },
    {
      case: 'an omitted function reference',
      function: undefined,
      expected: { name: 'Function unavailable', slug: '' },
    },
  ])('maps $case', ({ function: functionReference, expected }) => {
    const row = restFunctionRunToTableRun({
      id: 'run-1',
      function: functionReference,
      app: { id: 'app' },
      status: 'RUNNING',
      queuedAt: '2026-08-31T10:00:00Z',
    });

    expect(row.function).toEqual(expected);
  });

  it('rejects missing required references', () => {
    expect(() =>
      restFunctionRunToTableRun({
        id: 'run-1',
        function: { id: 'fn-id', name: 'Function' },
        status: 'RUNNING',
        queuedAt: '2026-08-31T10:00:00Z',
      }),
    ).toThrow('missing required');
    expect(() =>
      restFunctionRunToTableRun({
        id: '',
        function: { id: 'fn-id', name: 'Function' },
        app: { id: 'app' },
        status: 'RUNNING',
        queuedAt: '2026-08-31T10:00:00Z',
      }),
    ).toThrow('missing required');
  });

  it('computes elapsed duration for an unfinished run', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-08-31T10:04:45Z'));

    const row = restFunctionRunToTableRun({
      id: 'run-1',
      function: { id: 'fn-id', name: 'Function' },
      app: { id: 'app' },
      status: 'RUNNING',
      queuedAt: '2026-08-31T10:01:00Z',
      startedAt: '2026-08-31T10:01:30Z',
    });

    expect(row.durationMS).toBe(195_000);
  });

  it('uses the parent slug when its name is omitted', () => {
    const row = restFunctionRunToTableRun({
      id: 'run-1',
      function: { id: 'fn-id', name: 'Function' },
      app: { id: 'app' },
      status: 'COMPLETED',
      queuedAt: '2026-08-31T10:00:00Z',
      deferredFrom: { functionSlug: 'parent-function' },
    });

    expect(row.deferredFrom?.[0]).toEqual({
      function: { name: 'parent-function', slug: 'parent-function' },
    });
  });
});

it('translates selected app IDs for the REST request', async () => {
  const apiFetch = vi
    .fn()
    .mockResolvedValue(
      new Response(
        JSON.stringify({ data: [], page: { hasMore: false, limit: 40 } }),
        { status: 200 },
      ),
    );
  const restAppIDs = getRestAppIDs(
    ['internal-app-id'],
    [{ id: 'internal-app-id', externalID: 'public-app-id' }],
  );
  expect(restAppIDs).toEqual(['public-app-id']);
  if (!restAppIDs) throw new Error('expected translated app IDs');

  await fetchRestRuns(
    apiFetch,
    {
      appIDs: restAppIDs,
      functionSlug: null,
      startTime: '2026-08-31T10:00:00Z',
      endTime: '2026-08-31T11:00:00Z',
      status: ['RUNNING'],
      timeField: 'STARTED_AT',
      celQuery: undefined,
      isDeferred: false,
      environmentSlug: 'production',
      functionAppID: null,
    },
    'next-page',
    new AbortController().signal,
  );

  expect(apiFetch).toHaveBeenCalledOnce();
  expect(apiFetch).toHaveBeenCalledWith(
    '/v2/runs?from=2026-08-31T10%3A00%3A00Z&timeField=STARTED_AT&order=DESC&limit=40&include=deferred_from&until=2026-08-31T11%3A00%3A00Z&cursor=next-page&isDeferred=false&status=RUNNING&appId=public-app-id',
    { signal: expect.any(AbortSignal) },
  );
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
