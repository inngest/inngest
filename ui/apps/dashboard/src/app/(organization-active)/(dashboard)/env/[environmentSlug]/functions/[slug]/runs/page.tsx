'use client';

import { useEffect, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import StatusFilter from '@inngest/components/Filter/StatusFilter';
import TimeFieldFilter from '@inngest/components/Filter/TimeFieldFilter';
import { SelectGroup } from '@inngest/components/Select/Select';
import {
  type FunctionRunStatus,
  type FunctionRunTimeField,
} from '@inngest/components/types/functionRun';
import { getTimestampDaysAgo, toMaybeDate } from '@inngest/components/utils/date';
import { RiLoopLeftLine } from '@remixicon/react';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { FunctionRunTimeFieldV2 } from '@/gql/graphql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { useSearchParam, useStringArraySearchParam } from '@/utils/useSearchParam';
import Page from '../../../runs/[runID]/page';
import RunsTable from './RunsTable';
import TimeFilter from './TimeFilter';
import { toRunStatuses, toTimeField } from './utils';

const GetRunsDocument = graphql(`
  query GetRuns(
    $environmentID: ID!
    $startTime: Time!
    $status: [FunctionRunStatus!]
    $timeField: FunctionRunTimeFieldV2
    $functionSlug: String!
  ) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: { from: $startTime, status: $status, timeField: $timeField, fnSlug: $functionSlug }
        orderBy: [{ field: QUEUED_AT, direction: DESC }]
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
      }
    }
  }
`);

const renderSubComponent = ({ id }: { id: string }) => {
  return (
    <div className="mx-5">
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
  const functionSlug = decodeURIComponent(params.slug);

  const [rawFilteredStatus, setFilteredStatus, removeFilteredStatus] =
    useStringArraySearchParam('filterStatus');
  const [rawTimeField = FunctionRunTimeFieldV2.QueuedAt, setTimeField] =
    useSearchParam('timeField');
  const [lastDays = '3', setLastDays] = useSearchParam('last');

  const timeField = toTimeField(rawTimeField) ?? FunctionRunTimeFieldV2.QueuedAt;

  /* TODO: Time params for absolute time filter */
  // const [fromTime, setFromTime] = useSearchParam('from');
  // const [untilTime, setUntilTime] = useSearchParam('until');

  /* TODO: When we have absolute time, the start date will be either coming from the date picker or the relative time */
  const [startTime, setStartTime] = useState<Date>(new Date());

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
    if (value.length > 0) {
      setFilteredStatus(value);
    } else {
      removeFilteredStatus();
    }
  }

  function handleTimeFieldChange(value: FunctionRunTimeField) {
    if (value.length > 0) {
      setTimeField(value);
    }
  }

  function handleDaysChange(value: string) {
    if (value) {
      setLastDays(value);
    }
  }

  const environment = useEnvironment();
  const res = useSkippableGraphQLQuery({
    query: GetRunsDocument,
    skip: !functionSlug,
    variables: {
      environmentID: environment.id,
      functionSlug,
      startTime: startTime.toISOString(),
      status: filteredStatus.length > 0 ? filteredStatus : null,
      timeField,
    },
  });

  if (res.error) {
    throw res.error;
  }

  const runsData = res.data?.environment.runs.edges;

  if (functionSlug && !runsData && !res.isLoading) {
    throw new Error('missing run');
  }

  {
    /* TODO: This is a temp parser */
  }
  const runs = runsData?.map((edge) => {
    const startedAt = toMaybeDate(edge.node.startedAt);
    let durationMS = null;
    if (startedAt) {
      durationMS = (toMaybeDate(edge.node.endedAt) ?? new Date()).getTime() - startedAt.getTime();
    }

    return {
      id: edge.node.id,
      queuedAt: edge.node.queuedAt,
      endedAt: edge.node.endedAt,
      durationMS,
      status: edge.node.status,
    };
  });

  return (
    <main className="h-full min-h-0 overflow-y-auto bg-white">
      <div className="flex items-center justify-between gap-2 bg-slate-50 px-8 py-2">
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
        isLoading={res.isLoading}
        renderSubComponent={renderSubComponent}
        getRowCanExpand={() => true}
      />
    </main>
  );
}
