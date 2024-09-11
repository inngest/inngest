import { useEffect, useMemo, useRef } from 'react';
//@ts-ignore
import debounce from 'lodash.debounce';

const useDebounce = (callback: () => void, delay: number = 1000) => {
  const ref = useRef<(() => void) | null>(null);

  useEffect(() => {
    ref.current = callback;
  }, [callback]);

  const debouncedCallback = useMemo(() => {
    const func = () => {
      ref.current?.();
    };

    return debounce(func, delay);
  }, []);

  return debouncedCallback;
};

export default useDebounce;
