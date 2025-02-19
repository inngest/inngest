import { useCallback, useEffect, useMemo, useState } from 'react';

/**
 * Get and set a boolean value in localStorage. Includes a flag to indicate if
 * the value has been hydrated, avoiding a flash of content.
 */
export function useBooleanLocalStorage(
  key: string,
  defaultValue: boolean
): {
  set: (value: boolean) => void;
  isReady: boolean;
  value: boolean;
} {
  // Will set to the stored value after reading from localStorage.
  const [value, setValue] = useState<boolean>(defaultValue);

  // Will be true when done reading from localStorage.
  const [isReady, setIsReady] = useState(false);

  // Read from localStorage.
  useEffect(() => {
    try {
      const item = window.localStorage.getItem(key);

      // Is null if it doesn't exist.
      if (item) {
        const parsed: unknown = JSON.parse(item);
        if (typeof parsed !== 'boolean') {
          throw new Error('value is not a boolean');
        }
        setValue(parsed);
      }
    } catch (error) {
      console.warn(`error reading localStorage key "${key}":`, error);
    } finally {
      setIsReady(true);
    }
  }, [key, defaultValue]);

  // Setter that handles in-memory and localStorage updates.
  const set = useCallback(
    (value: boolean) => {
      try {
        setValue(value);
        window.localStorage.setItem(key, JSON.stringify(value));
      } catch (error) {
        console.warn(`error setting localStorage key "${key}":`, error);
      }
    },
    [key]
  );

  // Return memoized object to avoid re-renders in consumers.
  return useMemo(() => {
    return {
      set,
      isReady,
      value,
    };
  }, [isReady, set, value]);
}
