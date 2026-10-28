import { useCallback, useRef } from 'react';

export const ACCOUNT_CONCURRENCY_SECTION_ID = 'account-concurrency';
export const BURST_CONCURRENCY_SECTION_ID = 'burst-concurrency';

export function useScrollIntoViewOnHash(id: string) {
  const scrolled = useRef(false);
  return useCallback(
    (el: HTMLElement | null) => {
      if (!el || scrolled.current || window.location.hash !== `#${id}`) {
        return;
      }
      scrolled.current = true;
      el.scrollIntoView();
    },
    [id],
  );
}
