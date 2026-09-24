import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import { formatSQL } from '@/components/Insights/InsightsSQLEditor/utils';
import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import type { QueryTemplate } from '@/components/Insights/types';

const DEEP_LINK_DEFAULT_NAME = 'Untitled query';

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
  const lastProcessedDeepLink = useRef<string>();
  const [processedDeepLink, setProcessedDeepLink] = useState<string>();

  const queryIdFromUrl =
    typeof search.query_id === 'string' ? search.query_id : undefined;
  const sqlFromUrl =
    typeof search.sql === 'string' && search.sql.length > 0
      ? search.sql
      : undefined;
  const nameFromUrl =
    typeof search.name === 'string' && search.name.length > 0
      ? search.name
      : undefined;
  const deepLinkKey = queryIdFromUrl
    ? `query:${queryIdFromUrl}`
    : sqlFromUrl
    ? `sql:${sqlFromUrl}\u0000${nameFromUrl ?? ''}`
    : undefined;

  // Handle query_id or sql whenever the current deep link changes.
  // Gated on isHydrated to ensure tab state has been restored from localStorage
  // before we attempt to create a deep-linked tab. Without this gate, the hydration
  // effect (in a parent component) can overwrite the tab created here, leaving
  // activeTabId pointing to a nonexistent tab and causing a blank screen.
  useEffect(() => {
    if (!isHydrated) return;

    if (!deepLinkKey) {
      lastProcessedDeepLink.current = undefined;
      setProcessedDeepLink(undefined);
      return;
    }
    if (lastProcessedDeepLink.current === deepLinkKey) return;

    if (queryIdFromUrl) {
      // Wait for saved queries to finish loading and have data
      if (isSavedQueriesFetching || !queries.data) return;

      lastProcessedDeepLink.current = deepLinkKey;
      setProcessedDeepLink(deepLinkKey);

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

    // If there's a sql param (and no query_id), open a new tab with that SQL.
    // The tab name comes from the optional `name` param so callers (e.g. the
    // Failed Functions "Open in Insights" button) can label the tab without
    // depending on a built-in Insights template existing.
    if (sqlFromUrl) {
      lastProcessedDeepLink.current = deepLinkKey;
      setProcessedDeepLink(deepLinkKey);

      // Synthesize a QueryTemplate so the tab manager's template branch picks
      // it up and honors the name (the snapshot branch ignores name and falls
      // back to the "untitled" label). formatSQL now uses the ClickHouse
      // dialect, so formatting the incoming SQL no longer mangles function
      // calls like `toUnixTimestamp64Milli(...)`.
      const template: QueryTemplate = {
        id: `deeplink-${ulid()}`,
        name: nameFromUrl ?? DEEP_LINK_DEFAULT_NAME,
        query: formatSQL(sqlFromUrl),
        explanation: '',
        templateKind: 'time',
      };
      actions.createTabFromQuery(template);

      // Clear the deep-link params from the URL so refresh/bookmark doesn't
      // respawn duplicate tabs.
      navigate({
        search: (prev: Record<string, unknown>) => {
          const next = { ...prev };
          delete next.sql;
          delete next.name;
          return next;
        },
        replace: true,
      });
    }
  }, [
    deepLinkKey,
    queryIdFromUrl,
    sqlFromUrl,
    nameFromUrl,
    queries.data,
    isSavedQueriesFetching,
    actions,
    isHydrated,
    navigate,
  ]);

  // Update URL when active tab changes
  useEffect(() => {
    if (!isHydrated) return;
    // Don't replace an incoming deep link while it is waiting for hydration or
    // saved-query data. A processed SQL link can safely proceed: its unsaved tab
    // has no query_id to synchronize.
    if (deepLinkKey && lastProcessedDeepLink.current !== deepLinkKey) {
      return;
    }

    const newQueryId = activeSavedQueryId;

    // Don't update if URL already has the correct query_id
    if (queryIdFromUrl === newQueryId) return;

    // Update URL without triggering navigation. Return prev unchanged when the
    // value matches so callers that short-circuit on same-reference don't replace.
    navigate({
      search: (prev: Record<string, unknown>) => {
        const prevQueryId =
          typeof prev.query_id === 'string' ? prev.query_id : undefined;
        if (prevQueryId === newQueryId) return prev;
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
  }, [activeSavedQueryId, deepLinkKey, queryIdFromUrl, navigate, isHydrated]);

  return Boolean(deepLinkKey && processedDeepLink !== deepLinkKey);
}
