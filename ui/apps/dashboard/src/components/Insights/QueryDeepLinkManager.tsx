'use client';

import { useEffect, useRef } from 'react';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { toast } from 'sonner';

import { useTabManagerActions } from './InsightsTabManager/TabManagerContext';
import { HOME_TAB } from './InsightsTabManager/constants';
import { useStoredQueries } from './QueryHelperPanel/StoredQueriesContext';

type QueryDeepLinkManagerProps = {
  activeSavedQueryId?: string;
  activeTabId: string;
  children: React.ReactNode;
};

export function QueryDeepLinkManager({
  activeSavedQueryId,
  activeTabId,
  children,
}: QueryDeepLinkManagerProps) {
  const [activeQueryIdParam, updateActiveQueryIdParam, removeActiveQueryIdParam] =
    useSearchParam('activeQueryId');

  const { initialDeepLinkIdRef, hasProcessedDeepLinkRef } = useProcessDeepLink(activeQueryIdParam);

  useEffect(() => {
    if (activeSavedQueryId !== undefined) {
      if (activeQueryIdParam !== activeSavedQueryId) updateActiveQueryIdParam(activeSavedQueryId);
      return;
    }

    if (activeQueryIdParam !== undefined) {
      const { shouldGuard } = guardAgainstActiveQueryIdParamFlash({
        activeQueryIdParam,
        hasProcessedDeepLink: hasProcessedDeepLinkRef.current,
        initialDeepLinkId: initialDeepLinkIdRef.current,
        isHomeTabActive: activeTabId === HOME_TAB.id,
      });
      if (shouldGuard) return;

      removeActiveQueryIdParam();
    }
  }, [
    activeQueryIdParam,
    activeSavedQueryId,
    activeTabId,
    removeActiveQueryIdParam,
    updateActiveQueryIdParam,
  ]);

  return children;
}

type UseProcessDeepLinkResult = {
  hasProcessedDeepLinkRef: React.MutableRefObject<boolean>;
  initialDeepLinkIdRef: React.MutableRefObject<string | undefined>;
};

function useProcessDeepLink(activeQueryIdParam: string | undefined): UseProcessDeepLinkResult {
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

type GuardArgs = {
  activeQueryIdParam: string | undefined;
  hasProcessedDeepLink: boolean;
  initialDeepLinkId: string | undefined;
  isHomeTabActive: boolean;
};

type GuardResult = { shouldGuard: boolean };

/**
 * Returns true when we should temporarily avoid removing the activeQueryId param
 * to prevent a visible URL flash while a deep-link is still being processed.
 *
 * Without this, since we land on the home tab, the 'activeQueryId' param would be
 * temporarily removed and then would re-appear once the deep-link opens a new tab.
 *
 * We check a number of conditions to ensure that we don't block legitimate changes
 * to the 'activeQueryId' parameter like opening a different saved query.
 */
function guardAgainstActiveQueryIdParamFlash(args: GuardArgs): GuardResult {
  const { activeQueryIdParam, hasProcessedDeepLink, initialDeepLinkId, isHomeTabActive } = args;

  if (hasProcessedDeepLink) return { shouldGuard: false };
  if (!isHomeTabActive) return { shouldGuard: false };
  if (activeQueryIdParam !== initialDeepLinkId) return { shouldGuard: false };

  // All guard conditions met; avoid removing the param to prevent URL flash.
  return { shouldGuard: true };
}
