import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useBooleanLocalStorage } from './useBooleanLocalStorage';

describe('useBooleanLocalStorage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('uses the default for an unset key', () => {
    const { result } = renderHook(() => useBooleanLocalStorage('warning', true));
    expect(result.current.value).toBe(true);
    expect(result.current.isReady).toBe(true);
  });

  it.each([true, false])('hydrates a saved %s value', (saved) => {
    localStorage.setItem('warning', JSON.stringify(saved));
    const { result } = renderHook(() => useBooleanLocalStorage('warning', !saved));
    expect(result.current.value).toBe(saved);
  });

  it('does not carry a dismissed value into a new unset key', () => {
    const { result, rerender } = renderHook(({ key }) => useBooleanLocalStorage(key, true), {
      initialProps: { key: 'warning:first' },
    });
    act(() => result.current.set(false));
    expect(localStorage.getItem('warning:first')).toBe('false');
    rerender({ key: 'warning:second' });
    expect(result.current.value).toBe(true);
    expect(localStorage.getItem('warning:second')).toBeNull();
    rerender({ key: 'warning:first' });
    expect(result.current.value).toBe(false);
  });

  it('uses the updated default when a key has no saved value', () => {
    const { result, rerender } = renderHook(
      ({ fallback }) => useBooleanLocalStorage('warning', fallback),
      { initialProps: { fallback: false } }
    );
    rerender({ fallback: true });
    expect(result.current.value).toBe(true);
  });

  it.each(['invalid JSON', '123', 'null'])('falls back for invalid stored data: %s', (stored) => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    localStorage.setItem('first', 'false');
    localStorage.setItem('second', stored);
    const { result, rerender } = renderHook(({ key }) => useBooleanLocalStorage(key, true), {
      initialProps: { key: 'first' },
    });
    rerender({ key: 'second' });
    expect(result.current.value).toBe(true);
    expect(result.current.isReady).toBe(true);
  });
  it('falls back when reading a new key throws', () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    localStorage.setItem('first', 'false');
    const { result, rerender } = renderHook(({ key }) => useBooleanLocalStorage(key, true), {
      initialProps: { key: 'first' },
    });
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('Storage unavailable');
    });
    rerender({ key: 'second' });
    expect(result.current.value).toBe(true);
    expect(result.current.isReady).toBe(true);
  });
  it('keeps a saved boolean when the default changes', () => {
    localStorage.setItem('warning', 'false');
    const { result, rerender } = renderHook(
      ({ fallback }) => useBooleanLocalStorage('warning', fallback),
      { initialProps: { fallback: false } }
    );
    rerender({ fallback: true });
    expect(result.current.value).toBe(false);
  });
});
