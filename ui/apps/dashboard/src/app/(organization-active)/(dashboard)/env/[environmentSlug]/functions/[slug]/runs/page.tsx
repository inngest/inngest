'use client';

import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import StatusFilter from '@inngest/components/Filter/StatusFilter';
import TimeFilter from '@inngest/components/Filter/TimeFilter';
// import { SelectGroup } from '@inngest/components/Select/Select';
import {
  type FunctionRunStatus,
  type FunctionRunTimeField,
} from '@inngest/components/types/functionRun';
import { RiLoopLeftLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { FunctionRunTimeFieldV2 } from '@/gql/graphql';
import { useSearchParam, useStringArraySearchParam } from '@/utils/useSearchParam';
import RunsTable from './RunsTable';
import { toRunStatuses, toTimeField } from './utils';

const TimeFilterDefault = FunctionRunTimeFieldV2.QueuedAt;

const GetRunsDocument = graphql(`
  query GetRuns($environmentID: ID!, $startTime: Time!, $status: [FunctionRunStatus!]) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: { from: $startTime, status: $status }
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

  const filteredStatus = useMemo(() => {
    return toRunStatuses(rawFilteredStatus ?? []);
  }, [rawFilteredStatus]);

  const timeField = useMemo(() => {
    if (!rawTimeField) {
      return TimeFilterDefault;
    }
    return toTimeField(rawTimeField);
  }, [rawTimeField]);

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

  const environment = useEnvironment();
  const [{ data, fetching: fetchingRuns }, refetch] = useQuery({
    query: GetRunsDocument,
    variables: {
      environmentID: environment.id,
      startTime: '2024-04-19T11:26:03.203Z',
      status: filteredStatus.length > 0 ? filteredStatus : null,
      timeField: timeField ?? TimeFilterDefault,
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
          {/* <SelectGroup> */}
          <TimeFilter
            selectedTimeField={timeField ?? TimeFilterDefault}
            onTimeFieldChange={handleTimeFieldChange}
          />
          {/* TODO: Add date filter here */}
          {/* </SelectGroup> */}
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
