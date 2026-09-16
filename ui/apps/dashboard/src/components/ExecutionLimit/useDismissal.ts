import { useCallback, useEffect, useState } from 'react';

import {
  isDismissalActive,
  readDismissedAt,
  writeDismissedAt,
  type DismissalSurface,
} from './executionLimit';

export function useDismissal(
  surface: DismissalSurface,
  accountID: string | undefined,
) {
  const [dismissedAt, setDismissedAt] = useState<number | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    if (!accountID) return;
    setDismissedAt(readDismissedAt(surface, accountID));
    setIsReady(true);
  }, [surface, accountID]);

  const dismiss = useCallback(() => {
    if (!accountID) return;
    const now = Date.now();
    writeDismissedAt(surface, accountID, now);
    setDismissedAt(now);
  }, [surface, accountID]);

  return {
    isReady,
    isDismissed: isDismissalActive(dismissedAt, Date.now()),
    dismiss,
  };
}
