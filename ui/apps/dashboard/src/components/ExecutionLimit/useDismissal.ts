import { useCallback, useEffect, useState } from 'react';

import {
  dismissalStorageKey,
  isDismissalActive,
  readDismissedAt,
  writeDismissedAt,
  type DismissalSurface,
  type UsageBand,
} from './executionLimit';

export function useDismissal(
  surface: DismissalSurface,
  accountID: string | undefined,
  band: UsageBand | undefined,
) {
  const key =
    accountID && band ? dismissalStorageKey(surface, accountID, band) : null;
  const [dismissal, setDismissal] = useState<{
    key: string;
    dismissedAt: number | null;
  } | null>(null);

  useEffect(() => {
    if (!key || !accountID || !band) return;
    setDismissal({
      key,
      dismissedAt: readDismissedAt(surface, accountID, band),
    });
  }, [key, surface, accountID, band]);

  const dismiss = useCallback(() => {
    if (!key || !accountID || !band) return;
    const now = Date.now();
    writeDismissedAt(surface, accountID, band, now);
    setDismissal({ key, dismissedAt: now });
  }, [key, surface, accountID, band]);

  const isReady = key !== null && dismissal?.key === key;

  return {
    isReady,
    isDismissed:
      isReady && isDismissalActive(dismissal.dismissedAt, Date.now()),
    dismiss,
  };
}
