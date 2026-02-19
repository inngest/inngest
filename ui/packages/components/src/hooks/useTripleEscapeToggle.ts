import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * Returns a boolean that toggles whenever the user presses Escape 3 times
 * within 1 second. Default value is `true`.
 */
export function useTripleEscapeToggle(defaultValue = true): boolean {
  const [value, setValue] = useState(defaultValue);
  const timestamps = useRef<number[]>([]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key !== 'Escape') return;

    const now = Date.now();
    timestamps.current.push(now);

    // Keep only the last 3 timestamps
    if (timestamps.current.length > 3) {
      timestamps.current = timestamps.current.slice(-3);
    }

    if (timestamps.current.length === 3 && now - timestamps.current[0]! < 1000) {
      setValue((prev) => !prev);
      timestamps.current = [];
    }
  }, []);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  return value;
}
