import {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useMemo,
  useState,
} from 'react';
import { InfiniteScrollTrigger } from '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useBooleanSearchParam,
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParams';
import { CombinedError, useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useGetTrigger } from '@/components/RunDetails/useGetTrigger';
import { RunsOrderByField } from '@/gql/graphql';
import { useFunction } from '@/queries/functions';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import { AccountConcurrencyBanner } from './AccountConcurrencyBanner';
import { AppFilterDocument, CountRunsDocument } from './queries';
import { decodeRunsFrontier, RunsAPIError } from './restRuns';
import { useRunsPagination } from './useRunsPagination';
import { toRunStatuses, toTimeField } from './utils';

export const DEFAULT_POLL_INTERVAL = 1000;

export type RefreshRunsRef = {
  refresh: () => void;
};

type FnProps = {
  functionSlug: string;
  scope: 'fn';
};

type EnvProps = {
  functionSlug?: undefined;
  scope: 'env';
};

type Props = FnProps | EnvProps;

const parseCelSearchError = (
  combinedError: CombinedError | Error | undefined,
) => {
  if (
    combinedError instanceof RunsAPIError &&
    (combinedError.code === 'expression_invalid' ||
      combinedError.code === 'query_too_long')
  ) {
    return combinedError;
  }
  if (!(combinedError instanceof CombinedError)) return;
  return combinedError?.graphQLErrors.find(
    (error) => error.extensions.code == 'expression_invalid',
  );
};

export const Runs = forwardRef<RefreshRunsRef, Props>(function Runs(
  { functionSlug, scope }: Props,
  ref,
) {
  const env = useEnvironment();

  const [{ data: functionData }] = useFunction({
    functionSlug: functionSlug ?? '',
    pause: scope !== 'fn',
  });

  const [appsRes] = useQuery({
    pause: scope === 'fn',
    query: AppFilterDocument,
    variables: { envSlug: env.slug },
  });

  const { booleanFlag } = useBooleanFlag();

  const { value: tracePreviewEnabled } = booleanFlag(
    'traces-preview',
    true,
    true,
  );
  const { value: restRunsEnabled } = booleanFlag(
    'rest-runs-table',
    false,
    true,
  );
  const [appIDs] = useStringArraySearchParam('filterApp');
  const [rawFilteredStatus] = useStringArraySearchParam('filterStatus');
  const [rawTimeField = RunsOrderByField.QueuedAt] =
    useSearchParam('timeField');
  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');
  const [search] = useSearchParam('search');
  const [excludeDeferred = false] = useBooleanSearchParam('excludeDeferred');

  const timeField = toTimeField(rawTimeField) ?? RunsOrderByField.QueuedAt;

  /* The start date comes from either the absolute start time or the relative time */
  const calculatedStartTime = useCalculatedStartTime({ lastDays, startTime });

  const getTrigger = useGetTrigger();
  const features = useAccountFeatures();

  const filteredStatus = useMemo(() => {
    return toRunStatuses(rawFilteredStatus ?? []);
  }, [rawFilteredStatus]);

  const environment = useEnvironment();

  const commonQueryVars = useMemo(
    () => ({
      appIDs: appIDs ?? null,
      environmentID: environment.id,
      functionSlug: functionSlug ?? null,
      startTime: calculatedStartTime.toISOString(),
      endTime: endTime ?? null,
      status: filteredStatus.length > 0 ? filteredStatus : null,
      timeField,
      celQuery: search,
      isDeferred: excludeDeferred ? false : null,
      environmentSlug: environment.slug,
      functionAppID: functionData?.workspace.workflow?.app.externalID ?? null,
    }),
    [
      appIDs,
      environment.id,
      functionSlug,
      calculatedStartTime,
      endTime,
      filteredStatus,
      timeField,
      search,
      excludeDeferred,
      environment.slug,
      functionData?.workspace.workflow?.app.externalID,
    ],
  );

  const restStatusesSupported =
    commonQueryVars.status?.every((status) =>
      ['QUEUED', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED'].includes(
        status,
      ),
    ) ?? true;
  const useREST =
    restRunsEnabled &&
    restStatusesSupported &&
    (scope === 'env' || commonQueryVars.functionAppID !== null);

  // Use the new hook to manage pagination
  const {
    runs,
    isLoadingInitial,
    isLoadingMore,
    hasNextPage,
    loadMore,
    reset,
    error: paginationError,
    progressiveSearch,
  } = useRunsPagination({
    commonQueryVars,
    tracePreviewEnabled,
    useREST,
  });

  const [countRes, countRefetch] = useQuery({
    pause: useREST && Boolean(search),
    query: CountRunsDocument,
    requestPolicy: 'network-only',
    variables: commonQueryVars,
  });

  const searchError = parseCelSearchError(paginationError || countRes.error);

  let totalCount = undefined;
  if (!countRes.fetching) {
    // Only set the total count if the count query has finished loading since we
    // don't want to render stale data
    totalCount = countRes.data?.environment.runs.totalCount;
  }

  const onScrollToTop = useCallback(() => {
    // Not needed with new hook, but keeping for compatibility
  }, []);

  // The concurrency banner doesn't poll, so it piggybacks on the same refresh
  // funnel the header button and the imperative ref both go through.
  const [refreshNonce, setRefreshNonce] = useState(0);

  const onRefresh = useCallback(() => {
    reset();
    if (!(useREST && search)) countRefetch();
    setRefreshNonce((n) => n + 1);
  }, [countRefetch, reset, search, useREST]);

  useImperativeHandle(ref, () => ({
    refresh: () => {
      onRefresh();
    },
  }));

  return (
    <RunsPage
      apps={appsRes.data?.env?.apps.map((app) => ({
        id: app.id,
        name: app.externalID,
      }))}
      banner={
        <AccountConcurrencyBanner refreshNonce={refreshNonce} scope={scope} />
      }
      data={runs}
      features={{
        history: features.data?.history ?? 7,
        tracesPreview: tracePreviewEnabled,
        isDeferred: true,
      }}
      hasMore={hasNextPage}
      isLoadingInitial={isLoadingInitial}
      isLoadingMore={isLoadingMore}
      onRefresh={onRefresh}
      onScrollToTop={onScrollToTop}
      getTrigger={getTrigger}
      functionIsPaused={functionData?.workspace.workflow?.isPaused ?? false}
      scope={scope}
      totalCount={totalCount}
      searchError={searchError}
      error={paginationError}
      progressiveSearch={
        progressiveSearch
          ? {
              ...progressiveSearch,
              searchedThrough: decodeRunsFrontier(
                progressiveSearch.cursor,
                timeField,
              )?.toLocaleString(),
              insightsHref: runsInsightsHref(environment.slug, commonQueryVars),
            }
          : undefined
      }
      pollInterval={DEFAULT_POLL_INTERVAL}
      infiniteScrollTrigger={(containerRef) => (
        <InfiniteScrollTrigger
          onIntersect={loadMore}
          hasMore={hasNextPage}
          isLoading={isLoadingInitial || isLoadingMore}
          root={containerRef}
        />
      )}
    />
  );
});

function runsInsightsHref(
  envSlug: string,
  vars: {
    celQuery?: string;
    startTime: string;
    endTime: string | null;
    appIDs: string[] | null;
    functionSlug: string | null;
  },
) {
  const params = new URLSearchParams({ from: vars.startTime });
  if (vars.endTime) params.set('until', vars.endTime);
  if (vars.celQuery) params.set('cel', vars.celQuery);
  if (vars.functionSlug) params.set('function', vars.functionSlug);
  for (const appID of vars.appIDs ?? []) params.append('app', appID);
  return `/env/${encodeURIComponent(envSlug)}/insights?${params.toString()}`;
}
