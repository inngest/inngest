// @vitest-environment jsdom

import { act } from 'react-dom/test-utils';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { InngestAPIFetch } from '@/queries/useInngestAPIFetch';

import { useProgressiveRuns } from './useRunsPagination';

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
