import { useEffect, useMemo, useRef } from 'react';
//@ts-ignore
import debounce from 'lodash.debounce';

const useDebounce = (callback: () => void) => {
  const ref = useRef<(() => void) | null>(null);

  useEffect(() => {
    ref.current = callback;
  }, [callback]);

  const debouncedCallback = useMemo(() => {
    const func = () => {
      ref.current?.();
    };

    return debounce(func, 1000);
  }, []);

  return debouncedCallback;
};

export default useDebounce;
