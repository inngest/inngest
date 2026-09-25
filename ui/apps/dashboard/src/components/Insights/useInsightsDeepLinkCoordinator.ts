import { useEffect, useReducer } from 'react';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import { formatSQL } from '@/components/Insights/InsightsSQLEditor/utils';
import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import {
  consumeSQLPrefillURL,
  syncSavedQueryURL,
  type InsightsDeepLinkIntent,
} from '@/components/Insights/insightsSearchParams';
import type { Tab } from '@/components/Insights/types';

const DEEP_LINK_DEFAULT_NAME = 'Untitled query';

type ExpectedActiveTab =
  | { kind: 'saved-query'; savedQueryId: string }
  | { kind: 'sql-prefill'; tabId: string };

export type DeepLinkCoordinatorStatus =
  | 'idle'
  | 'waiting-for-hydration/resources'
  | 'applying'
  | 'applied'
  | 'error';

type CoordinatorState =
  | { status: 'idle' }
  | {
      status: 'waiting-for-hydration/resources';
      navigationKey: string;
    }
  | {
      status: 'applying';
      navigationKey: string;
      expectedActiveTab: ExpectedActiveTab;
    }
  | { status: 'applied'; navigationKey: string }
  | { status: 'error'; navigationKey: string };

type CoordinatorEvent =
  | { type: 'reset' }
  | {
      type: 'applying';
      navigationKey: string;
      expectedActiveTab: ExpectedActiveTab;
    }
  | { type: 'applied'; navigationKey: string }
  | { type: 'error'; navigationKey: string };

interface UseInsightsDeepLinkCoordinatorParams {
  actions: TabManagerActions;
  activeTab: Tab | undefined;
  currentHref: string;
  intent: InsightsDeepLinkIntent | undefined;
  isHydrated: boolean;
  navigate: (options: { href: string; replace?: boolean }) => void;
}

export function useInsightsDeepLinkCoordinator({
  actions,
  activeTab,
  currentHref,
  intent,
  isHydrated,
  navigate,
}: UseInsightsDeepLinkCoordinatorParams): {
  isPending: boolean;
  status: DeepLinkCoordinatorStatus;
} {
  const { queries, isSavedQueriesFetching } = useStoredQueries();
  const [storedLifecycle, dispatch] = useReducer(coordinatorReducer, {
    status: 'idle',
  });
  const navigationKey = intentKey(intent);
  const lifecycle = lifecycleForNavigation(storedLifecycle, navigationKey);

  // Resolve and apply inbound intent. Applying is complete only after the
  // tab manager's committed active tab matches the requested destination.
  useEffect(() => {
    if (lifecycle.status === 'idle') {
      if (storedLifecycle.status !== 'idle') dispatch({ type: 'reset' });
      return;
    }

    if (!intent || !navigationKey) return;

    if (lifecycle.status === 'waiting-for-hydration/resources') {
      if (!isHydrated) return;

      if (intent.kind === 'saved-query') {
        if (activeTab?.savedQueryId === intent.id) {
          const href = syncSavedQueryURL(currentHref, intent.id);
          if (href && href !== currentHref) navigate({ href, replace: true });
          dispatch({ type: 'applied', navigationKey });
          return;
        }
        if (isSavedQueriesFetching) return;

        const savedQuery = queries.data?.find(
          (query) => query.id === intent.id,
        );
        if (!savedQuery) {
          toast.error(
            queries.error
              ? 'Unable to load saved queries; please try again'
              : 'Unable to load query; please ensure that you have access to it',
          );
          dispatch({ type: 'error', navigationKey });
          // An unresolved query_id would otherwise reproduce the failure on
          // every reload. Remove it and let committed active-tab state become
          // the next declarative URL, if that tab is itself saved.
          const href = syncSavedQueryURL(currentHref, undefined);
          if (href) navigate({ href, replace: true });
          return;
        }

        const tab: Tab = {
          id: ulid(),
          name: savedQuery.name,
          query: savedQuery.sql,
          savedQueryId: savedQuery.id,
        };
        dispatch({
          type: 'applying',
          navigationKey,
          expectedActiveTab: {
            kind: 'saved-query',
            savedQueryId: savedQuery.id,
          },
        });
        actions.openQueryTab(tab);
        return;
      }

      const tab: Tab = {
        id: ulid(),
        name: intent.name ?? DEEP_LINK_DEFAULT_NAME,
        query: formatSQL(intent.sql),
      };
      dispatch({
        type: 'applying',
        navigationKey,
        expectedActiveTab: { kind: 'sql-prefill', tabId: tab.id },
      });
      actions.openQueryTab(tab);
      return;
    }

    if (
      lifecycle.status === 'applying' &&
      isExpectedTabActive(lifecycle.expectedActiveTab, activeTab)
    ) {
      // SQL-prefill is a one-shot command. Consume it only after its unsaved
      // tab is observably active; saved query_id remains declarative URL state.
      if (intent.kind === 'sql-prefill') {
        const href = consumeSQLPrefillURL(currentHref);
        if (href !== currentHref) navigate({ href, replace: true });
      } else {
        const href = syncSavedQueryURL(currentHref, intent.id);
        if (href && href !== currentHref) navigate({ href, replace: true });
      }
      dispatch({ type: 'applied', navigationKey });
    }
  }, [
    actions,
    activeTab,
    currentHref,
    intent,
    isHydrated,
    isSavedQueriesFetching,
    lifecycle,
    navigate,
    navigationKey,
    queries.data,
    queries.error,
    storedLifecycle.status,
  ]);

  // Outbound URL synchronization only runs from committed state. An inbound
  // intent that is waiting, applying, or terminally failed exclusively owns
  // the URL until it either commits or is removed.
  useEffect(() => {
    if (!isHydrated) return;
    if (
      lifecycle.status === 'waiting-for-hydration/resources' ||
      lifecycle.status === 'applying' ||
      lifecycle.status === 'error' ||
      intent?.kind === 'sql-prefill'
    ) {
      return;
    }

    const activeSavedQueryId = activeTab?.savedQueryId;
    if (intent?.kind === 'saved-query' && intent.id === activeSavedQueryId) {
      return;
    }
    if (!intent && !activeSavedQueryId) return;

    const href = syncSavedQueryURL(currentHref, activeSavedQueryId);
    if (href) navigate({ href, replace: true });
  }, [
    activeTab?.savedQueryId,
    currentHref,
    intent,
    isHydrated,
    lifecycle.status,
    navigate,
  ]);

  return {
    isPending:
      lifecycle.status === 'waiting-for-hydration/resources' ||
      lifecycle.status === 'applying',
    status: lifecycle.status,
  };
}

function coordinatorReducer(
  _state: CoordinatorState,
  event: CoordinatorEvent,
): CoordinatorState {
  switch (event.type) {
    case 'reset':
      return { status: 'idle' };
    case 'applying':
      return {
        status: 'applying',
        navigationKey: event.navigationKey,
        expectedActiveTab: event.expectedActiveTab,
      };
    case 'applied':
      return { status: 'applied', navigationKey: event.navigationKey };
    case 'error':
      return { status: 'error', navigationKey: event.navigationKey };
  }
}

function lifecycleForNavigation(
  state: CoordinatorState,
  navigationKey: string | undefined,
): CoordinatorState {
  if (!navigationKey) return { status: 'idle' };
  if ('navigationKey' in state && state.navigationKey === navigationKey) {
    return state;
  }
  return { status: 'waiting-for-hydration/resources', navigationKey };
}

function intentKey(
  intent: InsightsDeepLinkIntent | undefined,
): string | undefined {
  if (!intent) return undefined;
  return intent.kind === 'saved-query'
    ? `saved-query:${intent.id}`
    : `sql-prefill:${intent.sql}\u0000${intent.name ?? ''}`;
}

function isExpectedTabActive(
  expected: ExpectedActiveTab,
  activeTab: Tab | undefined,
): boolean {
  return expected.kind === 'saved-query'
    ? activeTab?.savedQueryId === expected.savedQueryId
    : activeTab?.id === expected.tabId;
}
