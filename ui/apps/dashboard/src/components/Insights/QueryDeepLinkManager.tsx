'use client';

import { useEffect, useRef } from 'react';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { toast } from 'sonner';

import { useTabManagerActions } from './InsightsTabManager/TabManagerContext';
import { useStoredQueries } from './QueryHelperPanel/StoredQueriesContext';

type QueryDeepLinkManagerProps = {
  activeSavedQueryId?: string;
  children: React.ReactNode;
};

export function QueryDeepLinkManager({ activeSavedQueryId, children }: QueryDeepLinkManagerProps) {
  const [activeQueryIdParam, updateActiveQueryIdParam, removeActiveQueryIdParam] =
    useSearchParam('activeQueryId');

  useEffect(() => {
    if (activeSavedQueryId !== undefined) {
      if (activeQueryIdParam !== activeSavedQueryId) updateActiveQueryIdParam(activeSavedQueryId);
      return;
    }

    if (activeQueryIdParam !== undefined) removeActiveQueryIdParam();
  }, [activeSavedQueryId, activeQueryIdParam, removeActiveQueryIdParam, updateActiveQueryIdParam]);

  useProcessInitialDeepLink(activeQueryIdParam);

  return children;
}

function useProcessInitialDeepLink(activeQueryIdParam: string | undefined) {
  const queryIdParamRef = useRef<string | undefined>(activeQueryIdParam);
  const { queries } = useStoredQueries();
  const { tabManagerActions } = useTabManagerActions();

  // Default to true if there's no deep-link to process.
  const hasProcessedDeepLink = useRef(queryIdParamRef.current === undefined);

  useEffect(() => {
    if (hasProcessedDeepLink.current) return;
    if (queries.isLoading) return;

    const targetId = queryIdParamRef.current;
    if (targetId === undefined) return;

    hasProcessedDeepLink.current = true;

    const matchingQuery = queries.data?.find((q) => q.id === targetId);

    if (matchingQuery !== undefined) {
      tabManagerActions.createTabFromQuery(matchingQuery);
      toast.success('Successfully loaded query.');
    } else {
      toast.error('Failed to load query.');
    }
  }, [queries.data, queries.isLoading, tabManagerActions]);
}
