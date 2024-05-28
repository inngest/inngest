'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import StatusFilter from '@inngest/components/Filter/StatusFilter';
import TimeFieldFilter from '@inngest/components/Filter/TimeFieldFilter';
import { SelectGroup } from '@inngest/components/Select/Select';
import { LoadingMore } from '@inngest/components/Table';
import {
  type FunctionRunStatus,
  type FunctionRunTimeField,
} from '@inngest/components/types/functionRun';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import { RiLoopLeftLine } from '@remixicon/react';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { RunsOrderByField } from '@/gql/graphql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { useSearchParam, useStringArraySearchParam } from '@/utils/useSearchParam';
import Page from '../../../runs/[runID]/page';
import RunsTable from './RunsTable';
import TimeFilter from './TimeFilter';
import { parseRunsData, toRunStatuses, toTimeField } from './utils';

const GetRunsDocument = graphql(`
  query GetRuns(
    $environmentID: ID!
    $startTime: Time!
    $status: [FunctionRunStatus!]
    $timeField: RunsOrderByField!
    $functionSlug: String!
    $functionRunCursor: String = null
  ) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: { from: $startTime, status: $status, timeField: $timeField, fnSlug: $functionSlug }
        orderBy: [{ field: $timeField, direction: DESC }]
        after: $functionRunCursor
      ) {
        edges {
          node {
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

const renderSubComponent = ({ id }: { id: string }) => {
  return (
    <div className="border-l-4 border-slate-400 px-5 pb-6">
      <Page params={{ runID: id }} />
    </div>
  );
};

export default function RunsPage({
  params,
}: {
  params: {
    slug: string;
  };
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const functionSlug = decodeURIComponent(params.slug);

  const [rawFilteredStatus, setFilteredStatus, removeFilteredStatus] =
    useStringArraySearchParam('filterStatus');
  const [rawTimeField = RunsOrderByField.QueuedAt, setTimeField] = useSearchParam('timeField');
  const [lastDays = '3', setLastDays] = useSearchParam('last');

  const timeField = toTimeField(rawTimeField) ?? RunsOrderByField.QueuedAt;

  /* TODO: Time params for absolute time filter */
  // const [fromTime, setFromTime] = useSearchParam('from');
  // const [untilTime, setUntilTime] = useSearchParam('until');

  /* TODO: When we have absolute time, the start date will be either coming from the date picker or the relative time */
  const [startTime, setStartTime] = useState<Date>(new Date());
  const [cursor, setCursor] = useState('');
  const [runs, setRuns] = useState<any[]>([]);
  const [isScrollRequest, setIsScrollRequest] = useState(false);

  useEffect(() => {
    if (lastDays) {
      setStartTime(
        getTimestampDaysAgo({
          currentDate: new Date(),
          days: parseInt(lastDays),
        })
      );
    }
  }, [lastDays]);

  const filteredStatus = useMemo(() => {
    return toRunStatuses(rawFilteredStatus ?? []);
  }, [rawFilteredStatus]);

  function handleStatusesChange(value: FunctionRunStatus[]) {
    scrollToTop();
    setIsScrollRequest(false);
    if (value.length > 0) {
      setFilteredStatus(value);
    } else {
      removeFilteredStatus();
    }
  }

  function handleTimeFieldChange(value: FunctionRunTimeField) {
    scrollToTop();
    setIsScrollRequest(false);
    if (value.length > 0) {
      setTimeField(value);
    }
  }

  function handleDaysChange(value: string) {
    scrollToTop();
    setIsScrollRequest(false);
    if (value) {
      setLastDays(value);
    }
  }

  const environment = useEnvironment();
  const firstPageRes = useSkippableGraphQLQuery({
    query: GetRunsDocument,
    skip: !functionSlug || isScrollRequest,
    variables: {
      environmentID: environment.id,
      functionSlug,
      startTime: startTime.toISOString(),
      status: filteredStatus.length > 0 ? filteredStatus : null,
      timeField,
      functionRunCursor: null,
    },
  });

  const nextPageRes = useSkippableGraphQLQuery({
    query: GetRunsDocument,
    skip: !functionSlug || !isScrollRequest,
    variables: {
      environmentID: environment.id,
      functionSlug,
      startTime: startTime.toISOString(),
      status: filteredStatus.length > 0 ? filteredStatus : null,
      timeField,
      functionRunCursor: cursor,
    },
  });

  if (firstPageRes.error || nextPageRes.error) {
    throw firstPageRes.error || nextPageRes.error;
  }

  const firstPageRunsData = firstPageRes.data?.environment.runs.edges;
  const nextPageRunsData = nextPageRes.data?.environment.runs.edges;
  const firstPageInfo = firstPageRes.data?.environment.runs.pageInfo;
  const nextPageInfo = nextPageRes.data?.environment.runs.pageInfo;

  if (functionSlug && !firstPageRunsData && !firstPageRes.isLoading && !firstPageRes.isSkipped) {
    throw new Error('missing run');
  }

  const firstPageRuns = useMemo(() => {
    return parseRunsData(firstPageRunsData);
  }, [firstPageRunsData]);

  const nextPageRuns = useMemo(() => {
    return parseRunsData(nextPageRunsData);
  }, [nextPageRunsData]);

  const scrollToTop = () => {
    if (containerRef.current) {
      containerRef.current.scrollTo({
        top: 0,
      });
    }
  };

  useEffect(() => {
    if (!isScrollRequest && firstPageRuns.length > 0) {
      setRuns(firstPageRuns);
    }
  }, [firstPageRuns, isScrollRequest]);

  useEffect(() => {
    if (isScrollRequest && nextPageRuns.length > 0) {
      setRuns((prevRuns) => [...prevRuns, ...nextPageRuns]);
    }
  }, [nextPageRuns, isScrollRequest]);

  const fetchMoreOnScroll = useCallback(
    (containerRefElement?: HTMLDivElement | null) => {
      if (containerRefElement && runs.length > 0) {
        const { scrollHeight, scrollTop, clientHeight } = containerRefElement;
        const lastCursor = nextPageInfo?.endCursor || firstPageInfo?.endCursor;
        const hasNextPage = nextPageInfo?.hasNextPage || firstPageInfo?.hasNextPage;
        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (
          reachedBottom &&
          !firstPageRes.isLoading &&
          !nextPageRes.isLoading &&
          lastCursor &&
          hasNextPage
        ) {
          setIsScrollRequest(true);
          setCursor(lastCursor);
        }
      }
    },
    [firstPageRes.isLoading, nextPageRes.isLoading, runs, nextPageInfo, firstPageInfo]
  );

  return (
    <main
      className="h-full min-h-0 overflow-y-auto bg-white"
      onScroll={(e) => fetchMoreOnScroll(e.target as HTMLDivElement)}
      ref={containerRef}
    >
      <div className="sticky top-0 flex items-center justify-between gap-2 bg-slate-50 px-8 py-2">
        <div className="flex items-center gap-2">
          <SelectGroup>
            <TimeFieldFilter
              selectedTimeField={timeField}
              onTimeFieldChange={handleTimeFieldChange}
            />
            <TimeFilter selectedDays={lastDays} onDaysChange={handleDaysChange} />
          </SelectGroup>
          <StatusFilter selectedStatuses={filteredStatus} onStatusesChange={handleStatusesChange} />
        </div>
        {/* TODO: wire button */}
        <Button
          label="Refresh"
          appearance="text"
          btnAction={() => {}}
          icon={<RiLoopLeftLine />}
          disabled
        />
      </div>
      <RunsTable
        //@ts-ignore
        data={runs}
        isLoading={firstPageRes.isLoading}
        renderSubComponent={renderSubComponent}
        getRowCanExpand={() => true}
      />
      {nextPageRes.isLoading && <LoadingMore />}
    </main>
  );
}
