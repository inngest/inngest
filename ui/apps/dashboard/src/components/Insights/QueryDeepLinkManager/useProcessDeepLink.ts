'use client';

import { useEffect, useRef } from 'react';
import { toast } from 'sonner';

import { useTabManagerActions } from '../InsightsTabManager/TabManagerContext';
import { useStoredQueries } from '../QueryHelperPanel/StoredQueriesContext';

export type UseProcessDeepLinkResult = {
  hasProcessedDeepLinkRef: React.MutableRefObject<boolean>;
  initialDeepLinkIdRef: React.MutableRefObject<string | undefined>;
};

export function useProcessDeepLink(
  activeQueryIdParam: string | undefined
): UseProcessDeepLinkResult {
  const initialDeepLinkIdRef = useRef<string | undefined>(activeQueryIdParam);
  const { queries } = useStoredQueries();
  const { tabManagerActions } = useTabManagerActions();

  // Default to true if there's no deep-link to process.
  const hasProcessedDeepLinkRef = useRef(initialDeepLinkIdRef.current === undefined);

  useEffect(() => {
    if (hasProcessedDeepLinkRef.current) return;
    if (queries.isLoading) return;

    const targetId = initialDeepLinkIdRef.current;
    if (targetId === undefined) return;

    hasProcessedDeepLinkRef.current = true;

    const matchingQuery = queries.data?.find((q) => q.id === targetId);

    if (matchingQuery !== undefined) {
      tabManagerActions.createTabFromQuery(matchingQuery);
      toast.success('Successfully loaded query.');
    } else {
      toast.error('Failed to load query.');
    }
  }, [queries.data, queries.isLoading, tabManagerActions]);

  return { hasProcessedDeepLinkRef, initialDeepLinkIdRef };
}
