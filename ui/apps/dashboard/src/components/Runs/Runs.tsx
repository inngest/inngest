import { forwardRef, useCallback, useImperativeHandle, useMemo } from 'react';
import { InfiniteScrollTrigger } from '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParams';
import { CombinedError, useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useGetTrigger } from '@/components/RunDetails/useGetTrigger';
import { GetFunctionPauseStateDocument, RunsOrderByField } from '@/gql/graphql';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import { AppFilterDocument, CountRunsDocument } from './queries';
import { useRunsPagination } from './useRunsPagination';
import { toRunStatuses, toTimeField } from './utils';

export const DEFAULT_POLL_INTERVAL = 0;

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

const parseCelSearchError = (combinedError: CombinedError | undefined) => {
  return combinedError?.graphQLErrors.find(
    (error) => error.extensions.code == 'expression_invalid',
  );
};

export const Runs = forwardRef<RefreshRunsRef, Props>(function Runs(
  { functionSlug, scope }: Props,
  ref,
) {
  const env = useEnvironment();

  const [{ data: pauseData }] = useQuery({
    pause: scope !== 'fn',
    query: GetFunctionPauseStateDocument,
    variables: {
      environmentID: env.id,
      functionSlug: functionSlug ?? '',
    },
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
    ],
  );

  // Use the new hook to manage pagination
  const {
    runs,
    isLoadingInitial,
    isLoadingMore,
    hasNextPage,
    loadMore,
    reset,
    error: paginationError,
  } = useRunsPagination({
    commonQueryVars,
    tracePreviewEnabled,
  });

  const [countRes] = useQuery({
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

  const onRefresh = useCallback(() => {
    reset();
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
      data={runs}
      features={{
        history: features.data?.history ?? 7,
        tracesPreview: tracePreviewEnabled,
      }}
      hasMore={hasNextPage}
      isLoadingInitial={isLoadingInitial}
      isLoadingMore={isLoadingMore}
      onRefresh={onRefresh}
      onScrollToTop={onScrollToTop}
      getTrigger={getTrigger}
      functionIsPaused={pauseData?.environment.function?.isPaused ?? false}
      scope={scope}
      totalCount={totalCount}
      searchError={searchError}
      error={paginationError}
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
