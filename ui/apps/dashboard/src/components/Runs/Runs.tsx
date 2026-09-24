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
import { decodeRunsFrontier, getRestAppIDs, RunsAPIError } from './restRuns';
import { runsInsightsHref } from './runsInsights';
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

const parseCelSearchError = (error: CombinedError | Error | undefined) => {
  // REST returns Error subclasses while URQL returns CombinedError. Check each
  // shape before reading transport-specific fields.
  if (
    error instanceof RunsAPIError &&
    (error.code === 'expression_invalid' || error.code === 'query_too_long')
  ) {
    return error;
  }
  if (!(error instanceof CombinedError)) return;
  return error.graphQLErrors.find(
    (item) => item.extensions.code === 'expression_invalid',
  );
};

export const Runs = forwardRef<RefreshRunsRef, Props>(function Runs(
  { functionSlug, scope }: Props,
  ref,
) {
  const env = useEnvironment();

  const [{ data: functionData, fetching: isFunctionLoading }] = useFunction({
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
  const { isReady: isRestRunsFlagReady, value: restRunsEnabled } = booleanFlag(
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
  const [forceRestRuns] = useBooleanSearchParam('forceRestRuns');

  const timeField = toTimeField(rawTimeField) ?? RunsOrderByField.QueuedAt;

  /* The start date comes from either the absolute start time or the relative time */
  const calculatedStartTime = useCalculatedStartTime({ lastDays, startTime });

  const getTrigger = useGetTrigger();
  const features = useAccountFeatures();

  const filteredStatus = useMemo(() => {
    return toRunStatuses(rawFilteredStatus ?? []);
  }, [rawFilteredStatus]);

  // TODO: Once REST is fully rolled out, store external app IDs in filterApp
  // and remove this translation, even though that will break old bookmarks.
  const restAppIDs = useMemo(
    () => getRestAppIDs(appIDs, appsRes.data?.env?.apps),
    [appIDs, appsRes.data?.env?.apps],
  );

  const environment = useEnvironment();

  const commonQueryVars = useMemo(
    () => ({
      appIDs: appIDs ?? null,
      restAppIDs,
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
      restAppIDs,
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

  const isRestRunsSelectionReady =
    forceRestRuns !== undefined || isRestRunsFlagReady;
  const isRestRunsRequested = forceRestRuns ?? restRunsEnabled;
  const isRestRunsMetadataLoading =
    isRestRunsRequested &&
    ((scope === 'env' && restAppIDs === undefined && appsRes.fetching) ||
      (scope === 'fn' &&
        commonQueryVars.functionAppID === null &&
        isFunctionLoading));
  const pauseRuns = !isRestRunsSelectionReady || isRestRunsMetadataLoading;
  const shouldUseREST =
    !pauseRuns &&
    isRestRunsRequested &&
    restAppIDs !== undefined &&
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
    shouldUseREST,
    pause: pauseRuns,
  });

  const [countRes, countRefetch] = useQuery({
    pause: pauseRuns || (shouldUseREST && Boolean(search)),
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
    if (!(shouldUseREST && search)) countRefetch();
    setRefreshNonce((n) => n + 1);
  }, [countRefetch, reset, search, shouldUseREST]);

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
              ),
              insightsHref: runsInsightsHref({
                envSlug: environment.slug,
                celQuery: search ?? '',
                startTime: commonQueryVars.startTime,
                endTime: commonQueryVars.endTime,
                timeField,
                appIDs: scope === 'env' ? restAppIDs : null,
                functionSlug: scope === 'fn' ? functionSlug : null,
              }),
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
