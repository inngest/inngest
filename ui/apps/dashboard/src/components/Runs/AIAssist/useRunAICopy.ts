import { useCallback, useEffect, useRef, useState } from 'react';

// Both copy actions need inline feedback rather than the shared helper's toast.
export function useRunAICopy(resetKey: string) {
  const [status, setStatus] = useState<
    'idle' | 'copying' | 'copied' | 'failed'
  >('idle');
  const attempt = useRef(0);
  const reset = useCallback(() => {
    attempt.current++;
    setStatus('idle');
  }, []);
  useEffect(() => {
    reset();
    return () => {
      attempt.current++;
    };
  }, [resetKey, reset]);
  useEffect(() => {
    if (status !== 'copied') return;
    const timer = setTimeout(reset, 2000);
    return () => clearTimeout(timer);
  }, [status, reset]);

  async function copy(
    text: string,
  ): Promise<'unavailable' | 'permission_denied' | 'other' | undefined> {
    const current = ++attempt.current;
    setStatus('copying');
    try {
      if (!navigator.clipboard?.writeText) {
        if (current === attempt.current) setStatus('failed');
        return 'unavailable';
      }
      await navigator.clipboard.writeText(text);
      if (current === attempt.current) setStatus('copied');
    } catch (error) {
      if (current === attempt.current) setStatus('failed');
      return error instanceof DOMException && error.name === 'NotAllowedError'
        ? 'permission_denied'
        : 'other';
    }
  }
  return { status, copy, reset };
}
