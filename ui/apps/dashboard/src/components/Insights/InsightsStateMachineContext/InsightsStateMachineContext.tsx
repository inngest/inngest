import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  type ReactNode,
} from 'react';
import { useQuery } from '@tanstack/react-query';

import { useStoredQueries } from '../QueryHelperPanel/StoredQueriesContext';
import { makeQuerySnapshot } from '../queries';
import { trackInsightsQueryRan } from '@/utils/analyticsEvents';
import type { InsightsQueryRunTrigger } from '../types';
import type { InsightsFetchResult, InsightsStatus } from './types';
import { useFetchInsights } from './useFetchInsights';

type RunQueryOptions = {
  trigger?: InsightsQueryRunTrigger;
};

interface InsightsStateMachineContextValue {
  data: InsightsFetchResult | undefined;
  error: null | Error;
  query: string;
  queryName: string;
  onChange: (value: string) => void;
  onNameChange: (name: string) => void;
  runQuery: (options?: RunQueryOptions) => void;
  status: InsightsStatus;
}

const InsightsStateMachineContext =
  createContext<InsightsStateMachineContextValue | null>(null);

type InsightsStateMachineContextProviderProps = {
  children: ReactNode;
  onAutoRunConsumed?: () => void;
  onQueryChange: (query: string) => void;
  onQueryNameChange: (name: string) => void;
  query: string;
  queryName: string;
  renderChildren: boolean;
  runOnMount?: boolean;
  savedQueryId?: string;
  tabId: string;
};

export function InsightsStateMachineContextProvider({
  children,
  onAutoRunConsumed,
  onQueryChange,
  onQueryNameChange,
  query,
  queryName,
  renderChildren,
  runOnMount,
  savedQueryId,
  tabId,
}: InsightsStateMachineContextProviderProps) {
  const { fetchInsights } = useFetchInsights();
  const { saveQuerySnapshot } = useStoredQueries();
  const runQueryTriggerRef = useRef<InsightsQueryRunTrigger>('unknown');

  const { data, error, isError, isFetching, refetch } = useQuery({
    enabled: false,
    gcTime: 0,
    queryKey: ['insights', tabId],
    queryFn: async () => {
      const trigger = runQueryTriggerRef.current;
      const startedAt =
        typeof performance === 'undefined' ? Date.now() : performance.now();
      const getDurationMs = () =>
        Math.round(
          (typeof performance === 'undefined'
            ? Date.now()
            : performance.now()) - startedAt,
        );

      try {
        const result = await fetchInsights(
          { query, queryName },
          (query, queryName) => {
            saveQuerySnapshot(makeQuerySnapshot(query, queryName));
          },
        );
        const hasDiagnosticErrors = result.diagnostics.some(
          (diagnostic) => diagnostic.severity === 'error',
        );

        trackInsightsQueryRan({
          feature: 'insights',
          data: result,
          durationMs: getDurationMs(),
          errorType: hasDiagnosticErrors ? 'diagnostic' : undefined,
          query,
          queryName,
          result: hasDiagnosticErrors ? 'failure' : 'success',
          savedQueryId,
          tabId,
          trigger,
        });

        return result;
      } catch (error) {
        trackInsightsQueryRan({
          feature: 'insights',
          durationMs: getDurationMs(),
          errorType: 'network',
          query,
          queryName,
          result: 'failure',
          savedQueryId,
          tabId,
          trigger,
        });
        throw error;
      } finally {
        runQueryTriggerRef.current = 'unknown';
      }
    },
    staleTime: 0,
    retry: false,
  });

  const runQuery = useCallback(
    (options: RunQueryOptions = {}) => {
      runQueryTriggerRef.current = options.trigger ?? 'unknown';
      refetch();
    },
    [refetch],
  );

  // Fire refetch() once per tab when a tab is created with a seeded query
  // that should execute automatically (e.g. the Failed Functions "Open in
  // Insights" deep link). Gated on `renderChildren` so inactive tabs don't
  // run, and on a ref so toggling the tab in/out of focus can't refire it.
  const hasAutoRunRef = useRef(false);
  useEffect(() => {
    if (hasAutoRunRef.current) return;
    if (!runOnMount || !renderChildren) return;

    hasAutoRunRef.current = true;
    refetch();
    onAutoRunConsumed?.();
  }, [runOnMount, renderChildren, refetch, onAutoRunConsumed]);

  return (
    <InsightsStateMachineContext.Provider
      value={{
        data,
        error,
        onChange: onQueryChange,
        onNameChange: onQueryNameChange,
        query,
        queryName,
        runQuery,
        status: getInsightsStatus({ data, isError, isLoading: isFetching }),
      }}
    >
      {renderChildren ? children : null}
    </InsightsStateMachineContext.Provider>
  );
}

export function useInsightsStateMachineContext() {
  const context = useContext(InsightsStateMachineContext);
  if (!context) {
    throw new Error(
      'useInsightsStateMachineContext must be used within InsightsStateMachineContextProvider',
    );
  }

  return context;
}

interface GetInsightsStatusParams {
  data: undefined | InsightsFetchResult;
  isError: boolean;
  isLoading: boolean;
}

export function getInsightsStatus({
  data,
  isError,
  isLoading,
}: GetInsightsStatusParams): InsightsStatus {
  if (isLoading) return 'loading';
  if (isError) return 'error';
  if (data?.diagnostics.find((x) => x.severity === 'error') !== undefined)
    return 'error';
  if (data !== undefined) return 'success';
  return 'initial';
}
