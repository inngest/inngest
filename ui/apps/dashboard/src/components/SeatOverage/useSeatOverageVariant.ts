'use client';

import { useSearchParams } from 'next/navigation';

export type SeatOverageVariant = 'banner' | 'toast';

export function useSeatOverageVariant(): SeatOverageVariant {
  const searchParams = useSearchParams();

  // Check for URL parameter to override variant for internal testing
  const variantParam = searchParams.get('seatOverageVariant');

  if (variantParam === 'toast') {
    return 'toast';
  }

  if (variantParam === 'banner') {
    return 'banner';
  }

  // Default to banner variant
  return 'banner';
}
