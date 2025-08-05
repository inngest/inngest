'use client';

import { useCallback, useEffect, useState } from 'react';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { parseSeatOverageData, useSeatOverageCheck } from './data';

const STORAGE_KEY = 'seatOverageDismissedUntil';

const shouldShowSeatCTA = (): boolean => {
  if (typeof window === 'undefined') return true;

  const until = localStorage.getItem(STORAGE_KEY);
  return !until || new Date(until) < new Date();
};

const dismissSeatCTA = (hours = 24): void => {
  if (typeof window === 'undefined') return;

  const until = new Date();
  until.setHours(until.getHours() + hours);
  localStorage.setItem(STORAGE_KEY, until.toISOString());
};

export function useSeatOverage() {
  const [isReady, setIsReady] = useState(false);
  const [shouldShow, setShouldShow] = useState(true);
  const trackingUser = useTrackingUser();

  const { data: rawData, error } = useSeatOverageCheck();
  const seatOverageData = parseSeatOverageData(rawData);

  useEffect(() => {
    setShouldShow(shouldShowSeatCTA());
    setIsReady(true);
  }, []);

  const dismiss = useCallback(
    (hours = 24) => {
      dismissSeatCTA(hours);
      setShouldShow(false);

      if (trackingUser) {
        trackEvent({
          name: 'app/upsell.seat.overage.dismissed',
          data: {
            variant: 'widget',
          },
          user: trackingUser,
          v: '2025-07-14.1',
        });
      }
    },
    [trackingUser]
  );

  const isWidgetVisible =
    !error && seatOverageData && seatOverageData.hasExceeded && isReady && shouldShow;

  return {
    isWidgetVisible,
    seatOverageData,
    error,
    dismiss,
  };
}
