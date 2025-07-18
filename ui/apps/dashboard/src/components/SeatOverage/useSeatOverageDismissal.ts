'use client';

import { useCallback, useEffect, useState } from 'react';

import { trackEvent, useTrackingUser } from '@/utils/tracking';

const STORAGE_KEY = 'seatOverageDismissedUntil';

export const shouldShowSeatCTA = (): boolean => {
  if (typeof window === 'undefined') return true;

  const until = localStorage.getItem(STORAGE_KEY);
  return !until || new Date(until) < new Date(); // expired or never set
};

export const dismissSeatCTA = (hours = 24): void => {
  if (typeof window === 'undefined') return;

  const until = new Date();
  until.setHours(until.getHours() + hours);
  localStorage.setItem(STORAGE_KEY, until.toISOString());
};

export function useSeatOverageDismissal() {
  const [isReady, setIsReady] = useState(false);
  const [shouldShow, setShouldShow] = useState(true);
  const trackingUser = useTrackingUser();

  // Check dismissal status on mount (SSR safe)
  useEffect(() => {
    setShouldShow(shouldShowSeatCTA());
    setIsReady(true);
  }, []);

  const dismiss = useCallback(
    (variant: 'banner' | 'toast', hours = 24) => {
      dismissSeatCTA(hours);
      setShouldShow(false);

      // Track dismissal event
      if (trackingUser) {
        trackEvent({
          name: 'app/upsell.seat.overage.dismissed',
          data: {
            variant,
          },
          user: trackingUser,
          v: '2025-07-14.1',
        });
      }
    },
    [trackingUser]
  );

  return {
    isReady,
    shouldShow,
    dismiss,
  };
}
