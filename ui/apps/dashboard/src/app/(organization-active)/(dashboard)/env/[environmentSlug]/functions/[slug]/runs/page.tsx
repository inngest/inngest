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
import { RunDetails } from '@/components/RunDetails/RunDetails';
import { graphql } from '@/gql';
import { RunsOrderByField } from '@/gql/graphql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { useSearchParam, useStringArraySearchParam } from '@/utils/useSearchParam';
import RunsTable, { type Run } from './RunsTable';
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
    <div className="border-subtle border-l-4 pb-6">
      <RunDetails standalone={false} runID={id} />
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
  const [runs, setRuns] = useState<Run[]>([]);
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

  function resetScrollPosition() {
    // This scroll cannot be smooth, has to be instantaneous
    scrollToTop();
    setIsScrollRequest(false);
  }

  function handleStatusesChange(value: FunctionRunStatus[]) {
    resetScrollPosition();
    if (value.length > 0) {
      setFilteredStatus(value);
    } else {
      removeFilteredStatus();
    }
  }

  function handleTimeFieldChange(value: FunctionRunTimeField) {
    resetScrollPosition();
    if (value.length > 0) {
      setTimeField(value);
    }
  }

  function handleDaysChange(value: string) {
    resetScrollPosition();
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
  const hasNextPage = nextPageInfo?.hasNextPage || firstPageInfo?.hasNextPage;
  const isLoading = firstPageRes.isLoading || nextPageRes.isLoading;

  if (functionSlug && !firstPageRunsData && !firstPageRes.isLoading && !firstPageRes.isSkipped) {
    throw new Error('missing run');
  }

  const firstPageRuns = useMemo(() => {
    return parseRunsData(firstPageRunsData);
  }, [firstPageRunsData]);

  const nextPageRuns = useMemo(() => {
    return parseRunsData(nextPageRunsData);
  }, [nextPageRunsData]);

  const scrollToTop = (smooth = false) => {
    if (containerRef.current) {
      containerRef.current.scrollTo({
        top: 0,
        behavior: smooth ? 'smooth' : 'auto',
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
        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (reachedBottom && !isLoading && lastCursor && hasNextPage) {
          setIsScrollRequest(true);
          setCursor(lastCursor);
        }
      }
    },
    [firstPageRes.isLoading, nextPageRes.isLoading, runs, nextPageInfo, firstPageInfo]
  );

  return (
    <main
      className="bg-canvasBase text-basis h-full min-h-0 overflow-y-auto"
      onScroll={(e) => fetchMoreOnScroll(e.target as HTMLDivElement)}
      ref={containerRef}
    >
      <div className="bg-canvasBase sticky top-0 z-[5] flex items-center justify-between gap-2 px-8 py-2">
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
        data={runs}
        isLoading={firstPageRes.isLoading}
        renderSubComponent={renderSubComponent}
        getRowCanExpand={() => true}
      />
      {nextPageRes.isLoading && <LoadingMore />}
      {!isLoading && !hasNextPage && (
        <div className="flex flex-col items-center py-8">
          <p className="text-subtle">No additional runs found.</p>
          <Button
            label="Back to top"
            kind="primary"
            appearance="text"
            btnAction={() => scrollToTop(true)}
          />
        </div>
      )}
    </main>
  );
}
