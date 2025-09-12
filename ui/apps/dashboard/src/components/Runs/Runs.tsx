'use client';

import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useState } from 'react';
import { InfiniteScrollTrigger } from '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
import type { Run } from '@inngest/components/RunsPage/types';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParam';
import { CombinedError, useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useGetTrigger } from '@/components/RunDetails/useGetTrigger';
import { GetFunctionPauseStateDocument, RunsOrderByField } from '@/gql/graphql';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import { AppFilterDocument, CountRunsDocument, GetRunsDocument } from './queries';
import { parseRunsData, toRunStatuses, toTimeField } from './utils';

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

const parseCelSearchError = (combinedError: CombinedError | undefined) => {
  return combinedError?.graphQLErrors.find(
    (error) => error.extensions.code == 'expression_invalid'
  );
};

export const Runs = forwardRef<RefreshRunsRef, Props>(function Runs(
  { functionSlug, scope }: Props,
  ref
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

  const { value: tracePreviewEnabled } = useBooleanFlag('traces-preview', false);

  const [appIDs] = useStringArraySearchParam('filterApp');
  const [rawFilteredStatus] = useStringArraySearchParam('filterStatus');
  const [rawTimeField = RunsOrderByField.QueuedAt] = useSearchParam('timeField');
  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');
  const [search] = useSearchParam('search');

  const timeField = toTimeField(rawTimeField) ?? RunsOrderByField.QueuedAt;

  /* The start date comes from either the absolute start time or the relative time */
  const calculatedStartTime = useCalculatedStartTime({ lastDays, startTime });
  const [cursor, setCursor] = useState('');
  const [runs, setRuns] = useState<Run[]>([]);
  const [isScrollRequest, setIsScrollRequest] = useState(false);

  const getTrigger = useGetTrigger();
  const features = useAccountFeatures();

  const filteredStatus = useMemo(() => {
    return toRunStatuses(rawFilteredStatus ?? []);
  }, [rawFilteredStatus]);

  const environment = useEnvironment();

  const commonQueryVars = {
    appIDs: appIDs ?? null,
    environmentID: environment.id,
    functionSlug: functionSlug ?? null,
    startTime: calculatedStartTime.toISOString(),
    endTime: endTime ?? null,
    status: filteredStatus.length > 0 ? filteredStatus : null,
    timeField,
    celQuery: search,
  };

  const [firstPageRes, fetchFirstPage] = useQuery({
    query: GetRunsDocument,
    pause: isScrollRequest,
    requestPolicy: 'network-only',
    variables: {
      ...commonQueryVars,
      functionRunCursor: null,
    },
  });

  const [nextPageRes] = useQuery({
    query: GetRunsDocument,
    pause: !isScrollRequest,
    requestPolicy: 'network-only',
    variables: {
      ...commonQueryVars,
      functionRunCursor: cursor,
    },
  });

  const [countRes] = useQuery({
    query: CountRunsDocument,
    pause: isScrollRequest,
    requestPolicy: 'network-only',
    variables: commonQueryVars,
  });

  const searchError = parseCelSearchError(
    firstPageRes.error || nextPageRes.error || countRes.error
  );

  const firstPageRunsData = firstPageRes.data?.environment.runs.edges;
  const nextPageRunsData = nextPageRes.data?.environment.runs.edges;
  const firstPageInfo = firstPageRes.data?.environment.runs.pageInfo;
  const nextPageInfo = nextPageRes.data?.environment.runs.pageInfo;
  const hasNextPage = isScrollRequest ? nextPageInfo?.hasNextPage : firstPageInfo?.hasNextPage;
  const isLoading = firstPageRes.fetching || nextPageRes.fetching;

  let totalCount = undefined;
  if (!countRes.fetching) {
    // Only set the total count if the count query has finished loading since we
    // don't want to render stale data
    totalCount = countRes.data?.environment.runs.totalCount;
  }

  if (functionSlug && !firstPageRunsData && !firstPageRes.fetching) {
    throw new Error('missing run');
  }

  const firstPageRuns = useMemo(() => {
    return parseRunsData(firstPageRunsData);
  }, [firstPageRunsData]);

  const nextPageRuns = useMemo(() => {
    return parseRunsData(nextPageRunsData);
  }, [nextPageRunsData]);

  useEffect(() => {
    if (!isScrollRequest) {
      setRuns(firstPageRuns);
    }
  }, [firstPageRuns, isScrollRequest]);

  useEffect(() => {
    if (isScrollRequest && nextPageRuns.length > 0) {
      setRuns((prevRuns) => [...prevRuns, ...nextPageRuns]);
    }
  }, [nextPageRuns, isScrollRequest]);

  const loadMore = useCallback(() => {
    if (runs.length > 0 && !isLoading && hasNextPage) {
      const lastCursor = nextPageInfo?.endCursor || firstPageInfo?.endCursor;
      if (lastCursor) {
        setIsScrollRequest(true);
        setCursor(lastCursor);
      }
    }
  }, [isLoading, hasNextPage, runs, nextPageInfo, firstPageInfo]);

  const onScrollToTop = useCallback(() => {
    setIsScrollRequest(false);
  }, []);

  const onRefresh = useCallback(() => {
    onScrollToTop();
    setCursor('');
    setRuns([]);
    fetchFirstPage();
  }, [fetchFirstPage, onScrollToTop]);

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
      hasMore={hasNextPage ?? false}
      isLoadingInitial={firstPageRes.fetching}
      isLoadingMore={nextPageRes.fetching}
      onRefresh={onRefresh}
      onScrollToTop={onScrollToTop}
      getTrigger={getTrigger}
      functionIsPaused={pauseData?.environment.function?.isPaused ?? false}
      scope={scope}
      totalCount={totalCount}
      searchError={searchError}
      infiniteScrollTrigger={
        <InfiniteScrollTrigger
          onIntersect={loadMore}
          hasMore={hasNextPage ?? false}
          isLoading={isLoading}
        />
      }
    />
  );
});
