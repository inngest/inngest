import { useCallback, useLayoutEffect, useRef } from 'react';

// Uses a ref to ensure that the latest value is always available.
export function useLatest<T>(value: T) {
  const r = useRef(value);

  useLayoutEffect(() => {
    r.current = value;
  }, [value]);

  return r;
}

// Extends useLatest to generate an always up-to-date callback.
export function useLatestCallback<A extends unknown[], R>(
  cb: (...args: A) => R,
) {
  const latest = useLatest(cb);

  return useCallback((...args: A) => {
    return latest.current(...args);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
}
