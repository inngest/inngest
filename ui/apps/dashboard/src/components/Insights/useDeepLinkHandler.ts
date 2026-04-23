import { useEffect, useRef } from 'react';
import { toast } from 'sonner';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { TEMPLATES } from '@/components/Insights/InsightsTabManager/InsightsTabPanelTemplatesTab/templates';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';

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
  const hasProcessedInitialQueryId = useRef(false);
  const hasProcessedInitialTemplateId = useRef(false);

  // Handle initial page load with query_id parameter.
  // Gated on isHydrated to ensure tab state has been restored from localStorage
  // before we attempt to create a deep-linked tab. Without this gate, the hydration
  // effect (in a parent component) can overwrite the tab created here, leaving
  // activeTabId pointing to a nonexistent tab and causing a blank screen.
  useEffect(() => {
    if (hasProcessedInitialQueryId.current) return;
    if (!isHydrated) return;

    const queryIdFromUrl =
      typeof search.query_id === 'string' ? search.query_id : undefined;
    if (!queryIdFromUrl) {
      hasProcessedInitialQueryId.current = true;
      return;
    }

    // Wait for saved queries to finish loading and have data
    if (isSavedQueriesFetching || !queries.data) return;

    // Mark as processed to prevent re-running
    hasProcessedInitialQueryId.current = true;

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
  }, [search, queries.data, isSavedQueriesFetching, actions, isHydrated]);

  // Handle initial page load with template_id parameter.
  // Opens a new tab seeded from a built-in template and strips the param from
  // the URL so refresh/bookmark doesn't keep spawning duplicate tabs.
  useEffect(() => {
    if (hasProcessedInitialTemplateId.current) return;
    if (!isHydrated) return;

    const templateIdFromUrl =
      typeof search.template_id === 'string' ? search.template_id : undefined;
    if (!templateIdFromUrl) {
      hasProcessedInitialTemplateId.current = true;
      return;
    }

    hasProcessedInitialTemplateId.current = true;

    const template = TEMPLATES.find((t) => t.id === templateIdFromUrl);
    if (template) {
      actions.createTabFromQuery(template, { runOnMount: true });
    } else {
      toast.error('Unable to load template; it may no longer exist');
    }

    navigate({
      search: (prev: Record<string, unknown>) => {
        const next = { ...prev };
        delete next.template_id;
        return next;
      },
      replace: true,
    });
  }, [search, actions, isHydrated, navigate]);

  // Update URL when active tab changes
  useEffect(() => {
    // Don't sync URL until we've processed the initial query_id and tabs are hydrated
    if (!hasProcessedInitialQueryId.current) return;
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
