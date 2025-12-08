import { useEffect, useRef } from "react";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import type { ReadonlyURLSearchParams } from "next/navigation";
import { toast } from "sonner";

import type { TabManagerActions } from "@/components/Insights/InsightsTabManager/InsightsTabManager";
import { useStoredQueries } from "@/components/Insights/QueryHelperPanel/StoredQueriesContext";

interface UseDeepLinkHandlerParams {
  actions: TabManagerActions;
  activeSavedQueryId: string | undefined;
  router: AppRouterInstance;
  searchParams: ReadonlyURLSearchParams;
}

export function useDeepLinkHandler({
  actions,
  activeSavedQueryId,
  router,
  searchParams,
}: UseDeepLinkHandlerParams) {
  const { queries, isSavedQueriesFetching } = useStoredQueries();
  const hasProcessedInitialQueryId = useRef(false);

  // Handle initial page load with query_id parameter
  useEffect(() => {
    if (hasProcessedInitialQueryId.current) return;

    const queryIdFromUrl = searchParams.get("query_id");
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
        "Unable to load query; please ensure that you have access to it",
      );
    }
  }, [searchParams, queries.data, isSavedQueriesFetching, actions]);

  // Update URL when active tab changes
  useEffect(() => {
    // Don't sync URL until we've processed the initial query_id
    if (!hasProcessedInitialQueryId.current) return;

    const currentQueryId = searchParams.get("query_id");
    const newQueryId = activeSavedQueryId;

    // Don't update if URL already has the correct query_id
    if (currentQueryId === newQueryId) return;

    const params = new URLSearchParams(searchParams.toString());

    if (newQueryId) {
      params.set("query_id", newQueryId);
    } else {
      params.delete("query_id");
    }

    // Update URL without triggering navigation
    const newUrl = params.toString()
      ? `?${params.toString()}`
      : window.location.pathname;
    router.replace(newUrl, { scroll: false });
  }, [activeSavedQueryId, searchParams, router]);
}
