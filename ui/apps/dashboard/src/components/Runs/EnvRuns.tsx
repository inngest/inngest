'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { RunsPage } from '@inngest/components/RunsPage/RunsPage';
import type { Run } from '@inngest/components/RunsPage/types';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParam';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useGetRun } from '@/components/RunDetails/useGetRun';
import { useGetTraceResult } from '@/components/RunDetails/useGetTraceResult';
import { useGetTrigger } from '@/components/RunDetails/useGetTrigger';
import { graphql } from '@/gql';
import { RunsOrderByField } from '@/gql/graphql';
import { useCancelRun } from '@/queries/useCancelRun';
import { useRerun } from '@/queries/useRerun';
import { pathCreator } from '@/utils/urls';
import { useGraphQLQuery, useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { usePlanFeatures } from '@/utils/usePlanFeatures';
import { parseRunsData, toRunStatuses, toTimeField } from './utils';

const GetRunsDocument = graphql(`
  query GetEnvRuns(
    $appIDs: [UUID!]
    $environmentID: ID!
    $startTime: Time!
    $endTime: Time
    $status: [FunctionRunStatus!]
    $timeField: RunsOrderByField!
    $functionRunCursor: String = null
  ) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: {
          appIDs: $appIDs
          from: $startTime
          until: $endTime
          status: $status
          timeField: $timeField
        }
        orderBy: [{ field: $timeField, direction: DESC }]
        after: $functionRunCursor
      ) {
        edges {
          node {
            app {
              externalID
              name
            }
            function {
              name
              slug
            }
            id
            queuedAt
            endedAt
            startedAt
            status
          }
        }
        pageInfo {
          hasNextPage
          hasPreviousPage
          startCursor
          endCursor
        }
      }
    }
  }
`);

const CountRunsDocument = graphql(`
  query CountEnvRuns(
    $appIDs: [UUID!]
    $environmentID: ID!
    $startTime: Time!
    $endTime: Time
    $status: [FunctionRunStatus!]
    $timeField: RunsOrderByField!
  ) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: {
          appIDs: $appIDs
          from: $startTime
          until: $endTime
          status: $status
          timeField: $timeField
        }
        orderBy: [{ field: $timeField, direction: DESC }]
      ) {
        totalCount
      }
    }
  }
`);

const AppFilterDocument = graphql(`
  query AppFilter($envSlug: String!) {
    env: envBySlug(slug: $envSlug) {
      apps {
        externalID
        id
        name
      }
    }
  }
`);

// TODO: DRY out this component along with FnRuns
export function EnvRuns() {
  const env = useEnvironment();

  const [appIDs] = useStringArraySearchParam('filterApp');
  const [rawFilteredStatus] = useStringArraySearchParam('filterStatus');
  const [rawTimeField = RunsOrderByField.QueuedAt] = useSearchParam('timeField');
  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');

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
  const features = usePlanFeatures();

  const appsRes = useGraphQLQuery({
    query: AppFilterDocument,
    variables: { envSlug: env.slug },
  });

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
    startTime: calculatedStartTime.toISOString(),
    endTime: endTime ?? null,
    status: filteredStatus.length > 0 ? filteredStatus : null,
    timeField,
  };

  const firstPageRes = useSkippableGraphQLQuery({
    query: GetRunsDocument,
    skip: isScrollRequest,
    variables: {
      ...commonQueryVars,
      functionRunCursor: null,
    },
  });

  const nextPageRes = useSkippableGraphQLQuery({
    query: GetRunsDocument,
    skip: !isScrollRequest,
    variables: {
      ...commonQueryVars,
      functionRunCursor: cursor,
    },
  });

  const countRes = useSkippableGraphQLQuery({
    query: CountRunsDocument,
    skip: isScrollRequest,
    variables: commonQueryVars,
  });

  if (firstPageRes.error || nextPageRes.error || countRes.error) {
    throw firstPageRes.error || nextPageRes.error || countRes.error;
  }

  const firstPageRunsData = firstPageRes.data?.environment.runs.edges;
  const nextPageRunsData = nextPageRes.data?.environment.runs.edges;
  const firstPageInfo = firstPageRes.data?.environment.runs.pageInfo;
  const nextPageInfo = nextPageRes.data?.environment.runs.pageInfo;
  const hasNextPage = nextPageInfo?.hasNextPage || firstPageInfo?.hasNextPage;
  const isLoading = firstPageRes.isLoading || nextPageRes.isLoading;

  let totalCount = undefined;
  if (!countRes.isLoading) {
    // Only set the total count if the count query has finished loading since we
    // don't want to render stale data
    totalCount = countRes.data?.environment.runs.totalCount;
  }

  if (!firstPageRunsData && !firstPageRes.isLoading && !firstPageRes.isSkipped) {
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
    [hasNextPage, isLoading, runs, nextPageInfo, firstPageInfo]
  );

  const onScrollToTop = useCallback(() => {
    setIsScrollRequest(false);
  }, []);

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
      isLoadingInitial={firstPageRes.isLoading}
      isLoadingMore={nextPageRes.isLoading}
      getRun={getRun}
      onScroll={fetchMoreOnScroll}
      onScrollToTop={onScrollToTop}
      getTraceResult={getTraceResult}
      getTrigger={getTrigger}
      pathCreator={internalPathCreator}
      rerun={rerun}
      scope="env"
      totalCount={totalCount}
    />
  );
}
