import { act, cleanup, renderHook, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';

import { useBooleanFlag } from './useBooleanFlag';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

// Guards like DuckDBGuard redirect as soon as isReady is true, so it can't
// flip before the flags themselves are loaded.
it('is only ready once the flags are parsed', async () => {
  let resolveBody!: (body: unknown) => void;
  const body = new Promise((resolve) => (resolveBody = resolve));
  const fetched = vi.fn().mockResolvedValue({ ok: true, json: () => body });
  vi.stubGlobal('fetch', fetched);

  const { result } = renderHook(() => useBooleanFlag('duckdb-insights'));
  await waitFor(() => expect(fetched).toHaveBeenCalled());
  await act(async () => {});
  expect(result.current).toEqual({ isReady: false, value: false });

  await act(async () => resolveBody({ features: { 'duckdb-insights': true } }));
  expect(result.current).toEqual({ isReady: true, value: true });
});

it('is ready w/ the default when the request fails', async () => {
  vi.spyOn(console, 'error').mockImplementation(() => {});
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }));

  const { result } = renderHook(() => useBooleanFlag('duckdb-insights', true));
  await waitFor(() => expect(result.current.isReady).toBe(true));
  expect(result.current.value).toBe(true);
});
