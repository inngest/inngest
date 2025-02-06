'use client';

import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useState } from 'react';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
import type { Run } from '@inngest/components/RunsPage/types';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParam';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useGetRun } from '@/components/RunDetails/useGetRun';
import { useGetTraceResult } from '@/components/RunDetails/useGetTraceResult';
import { useGetTrigger } from '@/components/RunDetails/useGetTrigger';
import { GetFunctionPauseStateDocument, RunsOrderByField } from '@/gql/graphql';
import { useCancelRun } from '@/queries/useCancelRun';
import { useRerun } from '@/queries/useRerun';
import { pathCreator } from '@/utils/urls';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import { useBooleanFlag } from '../FeatureFlags/hooks';
import { AppFilterDocument, CountRunsDocument, GetRunsDocument } from './queries';
import { parseRunsData, toRunStatuses, toTimeField } from './utils';

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
  ref
) {
  const env = useEnvironment();
  const { value: traceAIEnabled, isReady } = useBooleanFlag('ai-traces');

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

  const cancelRun = useCancelRun({ envID: env.id });
  const rerun = useRerun({ envID: env.id, envSlug: env.slug });
  const getTraceResult = useGetTraceResult();
  const getTrigger = useGetTrigger();
  const getRun = useGetRun();
  const features = useAccountFeatures();

  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      app: (params: { externalAppID: string }) =>
        pathCreator.app({ envSlug: env.slug, externalAppID: params.externalAppID }),
      function: (params: { functionSlug: string }) =>
        pathCreator.function({ envSlug: env.slug, functionSlug: params.functionSlug }),
      runPopout: (params: { runID: string }) =>
        pathCreator.runPopout({ envSlug: env.slug, runID: params.runID }),
    };
  }, [env.slug]);

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

  if (firstPageRes.error || nextPageRes.error || countRes.error) {
    throw firstPageRes.error || nextPageRes.error || countRes.error;
  }

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

  const fetchMoreOnScroll: React.ComponentProps<typeof RunsPage>['onScroll'] = useCallback(
    (event) => {
      if (runs.length > 0) {
        const { scrollHeight, scrollTop, clientHeight } = event.target as HTMLDivElement;
        const lastCursor = nextPageInfo?.endCursor || firstPageInfo?.endCursor;
        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (reachedBottom && !isLoading && lastCursor && hasNextPage) {
          setIsScrollRequest(true);
          setCursor(lastCursor);
        }
      }
    },
    [isLoading, hasNextPage, runs, nextPageInfo, firstPageInfo]
  );

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
      cancelRun={cancelRun}
      data={runs}
      features={{
        history: features.data?.history ?? 7,
      }}
      hasMore={hasNextPage ?? false}
      isLoadingInitial={firstPageRes.fetching}
      isLoadingMore={nextPageRes.fetching}
      getRun={getRun}
      onRefresh={onRefresh}
      onScroll={fetchMoreOnScroll}
      onScrollToTop={onScrollToTop}
      getTraceResult={getTraceResult}
      getTrigger={getTrigger}
      pathCreator={internalPathCreator}
      rerun={rerun}
      functionIsPaused={pauseData?.environment.function?.isPaused ?? false}
      scope={scope}
      totalCount={totalCount}
      traceAIEnabled={isReady && traceAIEnabled}
    />
  );
});
