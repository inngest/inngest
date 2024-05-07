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
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import { RiLoopLeftLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { FunctionRunTimeFieldV2 } from '@/gql/graphql';
import { useSearchParam, useStringArraySearchParam } from '@/utils/useSearchParam';
import RunsTable from './RunsTable';
import TimeFilter from './TimeFilter';
import { toRunStatuses, toTimeField } from './utils';

const GetRunsDocument = graphql(`
  query GetRuns(
    $environmentID: ID!
    $startTime: Time!
    $status: [FunctionRunStatus!]
    $timeField: FunctionRunTimeFieldV2
  ) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: { from: $startTime, status: $status, timeField: $timeField }
        orderBy: [{ field: QUEUED_AT, direction: ASC }]
      ) {
        edges {
          node {
            id
            queuedAt
            endedAt
            durationMS
            status
          }
        }
      }
    }
  }
`);

const renderSubComponent = ({ id }: { id: string }) => {
  /* TODO: Render the timeline instead */
  return <p>Subrow {id}</p>;
};

export default function RunsPage() {
  const [rawFilteredStatus, setFilteredStatus, removeFilteredStatus] =
    useStringArraySearchParam('filterStatus');
  const [rawTimeField, setTimeField] = useSearchParam('timeField');
  const [lastDays = '3', setLastDays] = useSearchParam('last');

  const timeField = toTimeField(rawTimeField ?? '') ?? FunctionRunTimeFieldV2.QueuedAt;

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
  const [{ data, fetching: fetchingRuns }, refetch] = useQuery({
    query: GetRunsDocument,
    variables: {
      environmentID: environment.id,
      startTime: startTime.toISOString(),
      status: filteredStatus.length > 0 ? filteredStatus : null,
      timeField,
    },
  });

  {
    /* TODO: This is a temp parser */
  }
  const runs = data?.environment.runs.edges.map((edge) => ({
    id: edge.node.id,
    queuedAt: edge.node.queuedAt,
    endedAt: edge.node.endedAt,
    durationMS: edge.node.durationMS,
    status: edge.node.status,
  }));

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
        <Button label="Refresh" appearance="text" btnAction={refetch} icon={<RiLoopLeftLine />} />
      </div>
      <RunsTable
        //@ts-ignore
        data={runs}
        isLoading={fetchingRuns}
        renderSubComponent={renderSubComponent}
        getRowCanExpand={() => true}
      />
    </main>
  );
}
