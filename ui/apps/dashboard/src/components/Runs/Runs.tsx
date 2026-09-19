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
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useGetTrigger } from '@/components/RunDetails/useGetTrigger';
import { RunsOrderByField } from '@/gql/graphql';
import { useFunction } from '@/queries/functions';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import { AccountConcurrencyBanner } from './AccountConcurrencyBanner';
import { AppFilterDocument } from './queries';
import { decodeRunsFrontier, getRestAppIDs, RunsAPIError } from './restRuns';
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

  // App filters and existing bookmarks store GraphQL app IDs, while the REST
  // endpoint accepts external app IDs.
  const restAppIDs = useMemo(
    () => getRestAppIDs(appIDs, appsRes.data?.env?.apps),
    [appIDs, appsRes.data?.env?.apps],
  );

  const commonQueryVars = useMemo(
    () => ({
      appIDs: restAppIDs ?? null,
      functionSlug: functionSlug ?? null,
      startTime: calculatedStartTime.toISOString(),
      endTime: endTime ?? null,
      status: filteredStatus.length > 0 ? filteredStatus : null,
      timeField,
      celQuery: search,
      isDeferred: excludeDeferred ? false : null,
      environmentSlug: env.slug,
      functionAppID: functionData?.workspace.workflow?.app.externalID ?? null,
    }),
    [
      restAppIDs,
      functionSlug,
      calculatedStartTime,
      endTime,
      filteredStatus,
      timeField,
      search,
      excludeDeferred,
      env.slug,
      functionData?.workspace.workflow?.app.externalID,
    ],
  );

  const runsEnabled =
    restAppIDs !== undefined &&
    (scope === 'env' || commonQueryVars.functionAppID !== null);

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
    enabled: runsEnabled,
  });
  const searchError =
    paginationError instanceof RunsAPIError &&
    (paginationError.code === 'expression_invalid' ||
      paginationError.code === 'query_too_long')
      ? paginationError
      : undefined;

  const onScrollToTop = useCallback(() => {
    // Not needed with new hook, but keeping for compatibility
  }, []);

  // The concurrency banner doesn't poll, so it piggybacks on the same refresh
  // funnel the header button and the imperative ref both go through.
  const [refreshNonce, setRefreshNonce] = useState(0);

  const onRefresh = useCallback(() => {
    reset();
    setRefreshNonce((n) => n + 1);
  }, [reset]);

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
      totalCount={undefined}
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
