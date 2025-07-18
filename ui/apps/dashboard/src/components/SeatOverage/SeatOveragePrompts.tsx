'use client';

import { SeatOverageBanner } from './SeatOverageBanner';
import { SeatOverageToast } from './SeatOverageToast';
import { parseSeatOverageData, useSeatOverageCheck } from './data';
import { useSeatOverageDismissal } from './useSeatOverageDismissal';
import { useSeatOverageVariant } from './useSeatOverageVariant';

export function SeatOveragePrompts() {
  const { data: rawData, error } = useSeatOverageCheck();
  const variant = useSeatOverageVariant();

  // Time-based dismissal system for both banner and toast
  const dismissal = useSeatOverageDismissal();

  // Parse the seat overage data
  const seatOverageData = parseSeatOverageData(rawData);

  // Don't show anything if:
  // - There's an error
  // - No seat overage data
  // - User hasn't exceeded the limit
  // - Dismissal localStorage isn't ready
  // - User has dismissed and it hasn't expired
  if (
    error ||
    !seatOverageData ||
    !seatOverageData.hasExceeded ||
    !dismissal.isReady ||
    !dismissal.shouldShow
  ) {
    return null;
  }

  // Render based on variant
  switch (variant) {
    case 'banner': {
      return (
        <SeatOverageBanner data={seatOverageData} onDismiss={() => dismissal.dismiss('banner')} />
      );
    }
    case 'toast': {
      return (
        <SeatOverageToast data={seatOverageData} onDismiss={() => dismissal.dismiss('toast')} />
      );
    }
    default: {
      return null;
    }
  }
}
