import { useCallback, useEffect, useRef } from 'react';
//@ts-ignore
import debounce from 'lodash.debounce';

const useDebounce = (callback: () => void, delay: number = 1000) => {
  const ref = useRef<(() => void) | null>(null);

  useEffect(() => {
    ref.current = callback;
  }, [callback]);

  const debouncedFunction = useRef(
    debounce(() => {
      ref.current?.();
    }, delay)
  ).current;

  return useCallback(debouncedFunction, [debouncedFunction]);
};

export default useDebounce;
