'use client';

import { useEffect, useRef } from 'react';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';

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

  useEffect(() => {
    if (queries.isLoading) return;

    const targetId = queryIdParamRef.current;
    if (targetId === undefined) return;

    const matchingQuery = queries.data?.find((q) => q.id === targetId);

    if (matchingQuery !== undefined) {
      console.log('[Insights deep-link] match found for saved query id:', targetId);
      return;
    }

    console.log('[Insights deep-link] no match for saved query id:', targetId);
  }, [queries.data, queries.isLoading]);
}
