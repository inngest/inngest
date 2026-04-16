import { useEffect, useRef } from 'react';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import type { QuerySnapshot } from '@/components/Insights/types';

interface UseDeepLinkHandlerParams {
  actions: TabManagerActions;
  activeSavedQueryId: string | undefined;
  isHydrated: boolean;
  navigate: (opts: {
    search: (prev: Record<string, unknown>) => Record<string, unknown>;
    replace?: boolean;
  }) => void;
  search: Record<string, unknown>;
}

export function useDeepLinkHandler({
  actions,
  activeSavedQueryId,
  isHydrated,
  navigate,
  search,
}: UseDeepLinkHandlerParams) {
  const { queries, isSavedQueriesFetching } = useStoredQueries();
  const hasProcessedInitialDeepLink = useRef(false);

  // Handle initial page load with query_id or sql parameter.
  // Gated on isHydrated to ensure tab state has been restored from localStorage
  // before we attempt to create a deep-linked tab. Without this gate, the hydration
  // effect (in a parent component) can overwrite the tab created here, leaving
  // activeTabId pointing to a nonexistent tab and causing a blank screen.
  useEffect(() => {
    if (hasProcessedInitialDeepLink.current) return;
    if (!isHydrated) return;

    const queryIdFromUrl =
      typeof search.query_id === 'string' ? search.query_id : undefined;
    const sqlFromUrl =
      typeof search.sql === 'string' && search.sql.length > 0
        ? search.sql
        : undefined;

    // If there's a query_id, handle it (takes precedence over sql)
    if (queryIdFromUrl) {
      // Wait for saved queries to finish loading and have data
      if (isSavedQueriesFetching || !queries.data) return;

      // Mark as processed to prevent re-running
      hasProcessedInitialDeepLink.current = true;

      // Check if the query exists
      const savedQuery = queries.data.find((q) => q.id === queryIdFromUrl);

      if (savedQuery) {
        // Programmatically open the tab
        actions.createTabFromQuery(savedQuery);
      } else {
        // Show error toast if query not found
        toast.error(
          'Unable to load query; please ensure that you have access to it',
        );
      }
      return;
    }

    // If there's a sql param (and no query_id), open a new tab with that SQL
    if (sqlFromUrl) {
      hasProcessedInitialDeepLink.current = true;

      const snapshot: QuerySnapshot = {
        id: ulid(),
        isSnapshot: true,
        name: 'Experiment Query',
        query: sqlFromUrl,
      };
      actions.createTabFromQuery(snapshot);

      // Clear the sql param from the URL to avoid re-opening on refresh
      navigate({
        search: (prev: Record<string, unknown>) => {
          const next = { ...prev };
          delete next.sql;
          return next;
        },
        replace: true,
      });
      return;
    }

    hasProcessedInitialDeepLink.current = true;
  }, [
    search,
    queries.data,
    isSavedQueriesFetching,
    actions,
    isHydrated,
    navigate,
  ]);

  // Update URL when active tab changes
  useEffect(() => {
    // Don't sync URL until we've processed the initial query_id and tabs are hydrated
    if (!hasProcessedInitialDeepLink.current) return;
    if (!isHydrated) return;

    const currentQueryId =
      typeof search.query_id === 'string' ? search.query_id : undefined;
    const newQueryId = activeSavedQueryId;

    // Don't update if URL already has the correct query_id
    if (currentQueryId === newQueryId) return;

    // Update URL without triggering navigation
    navigate({
      search: (prev: Record<string, unknown>) => {
        const next = { ...prev };
        if (newQueryId) {
          next.query_id = newQueryId;
        } else {
          delete next.query_id;
        }
        return next;
      },
      replace: true,
    });
  }, [activeSavedQueryId, search, navigate, isHydrated]);
}
