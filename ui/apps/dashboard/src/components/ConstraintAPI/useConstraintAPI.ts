import { useCallback, useEffect, useMemo, useState } from 'react';
import { useQuery } from 'urql';

import {
  ConstraintAPIEnrollmentQuery,
  ConstraintAPIInEffectQuery,
  parseConstraintAPIData,
} from './data';

const STORAGE_KEY = 'constraintAPIWidgetDismissedUntil';

export function useConstraintAPI() {
  const [isReady, setIsReady] = useState(false);
  const [shouldShow, setShouldShow] = useState(true);

  // Dual queries for enrollment status
  const [enrolledResult] = useQuery({
    query: ConstraintAPIEnrollmentQuery,
  });

  const [inEffectResult] = useQuery({
    query: ConstraintAPIInEffectQuery,
  });

  const constraintAPIData = useMemo(() => {
    if (enrolledResult.data && inEffectResult.data) {
      return parseConstraintAPIData(enrolledResult.data, inEffectResult.data);
    }
    return null;
  }, [enrolledResult.data, inEffectResult.data]);

  useEffect(() => {
    // Check localStorage for dismissal timestamp
    if (typeof window === 'undefined') {
      setIsReady(true);
      return;
    }

    const dismissedUntil = localStorage.getItem(STORAGE_KEY);
    if (dismissedUntil) {
      const now = new Date().getTime();
      setShouldShow(now > parseInt(dismissedUntil, 10));
    }
    setIsReady(true);
  }, []);

  const dismiss = useCallback(() => {
    if (typeof window === 'undefined') return;

    const hoursFromNow = new Date().getTime() + 24 * 60 * 60 * 1000;
    localStorage.setItem(STORAGE_KEY, hoursFromNow.toString());
    setShouldShow(false);
  }, []);

  // Widget visible if: ready, not dismissed, no errors, has data
  const isWidgetVisible =
    isReady &&
    shouldShow &&
    !enrolledResult.error &&
    !inEffectResult.error &&
    constraintAPIData !== null;

  return {
    isWidgetVisible,
    constraintAPIData,
    dismiss,
    isLoading: enrolledResult.fetching || inEffectResult.fetching,
    error: enrolledResult.error || inEffectResult.error,
  };
}
