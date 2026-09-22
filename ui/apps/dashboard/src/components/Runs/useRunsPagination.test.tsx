// @vitest-environment jsdom

import { act } from 'react-dom/test-utils';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { InngestAPIFetch } from '@/queries/useInngestAPIFetch';

import { fetchRestRuns, useProgressiveRuns } from './useRunsPagination';

(
  globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const baseVars = {
  appIDs: null,
  restAppIDs: null,
  environmentID: 'environment-id',
  functionSlug: null,
  startTime: '2026-09-15T00:00:00Z',
  endTime: null,
  status: null,
  timeField: 'STARTED_AT',
  isDeferred: null,
  environmentSlug: 'production',
  functionAppID: null,
};

describe('fetchRestRuns', () => {
  it.each([
    ['app-test-fn', 'test-fn'],
    ['app-app-test-fn', 'app-test-fn'],
  ])(
    'sends the configured ID for stored function slug %s',
    async (functionSlug, expectedFunctionID) => {
      const apiFetch = vi.fn<InngestAPIFetch>(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              data: [],
              page: { hasMore: false, limit: 40 },
            }),
            { status: 200 },
          ),
        ),
      );

      await fetchRestRuns(
        apiFetch,
        {
          ...baseVars,
          celQuery: undefined,
          functionSlug,
          functionAppID: 'app',
        },
        undefined,
        new AbortController().signal,
      );

      expect(apiFetch).toHaveBeenCalledOnce();
      expect(apiFetch.mock.calls[0]?.[0]).toMatch(
        new RegExp(`/v2/apps/app/functions/${expectedFunctionID}/runs\\?`),
      );
    },
  );
});

function ProgressiveRunsHarness({
  apiFetch,
  celQuery,
}: {
  apiFetch: InngestAPIFetch;
  celQuery: string;
}) {
  useProgressiveRuns({
    enabled: true,
    apiFetch,
    vars: { ...baseVars, celQuery },
  });
  return null;
}

describe('useProgressiveRuns', () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(async () => {
    if (root) {
      await act(async () => root?.unmount());
    }
    container?.remove();
    root = undefined;
    container = undefined;
  });

  it('starts one fresh request when the search input changes', async () => {
    const apiFetch = vi.fn<InngestAPIFetch>((_pathname, init) => {
      return new Promise<Response>((_resolve, reject) => {
        init?.signal?.addEventListener(
          'abort',
          () => reject(new DOMException('Aborted', 'AbortError')),
          { once: true },
        );
      });
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root?.render(
        <ProgressiveRunsHarness
          apiFetch={apiFetch}
          celQuery="event.data.first == true"
        />,
      );
    });
    expect(apiFetch).toHaveBeenCalledTimes(1);

    await act(async () => {
      root?.render(
        <ProgressiveRunsHarness
          apiFetch={apiFetch}
          celQuery="event.data.second == true"
        />,
      );
    });

    expect(apiFetch).toHaveBeenCalledTimes(2);
    expect(apiFetch.mock.calls[1]?.[0]).toContain(
      'query=event.data.second+%3D%3D+true',
    );
    expect(apiFetch.mock.calls[1]?.[0]).not.toContain('cursor=');
  });
});
