'use client';

import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import StatusFilter from '@inngest/components/Filter/StatusFilter';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { RiLoopLeftLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { useStringArraySearchParam } from '@/utils/useSearchParam';
import RunsTable from './RunsTable';
import { toRunStatuses } from './utils';

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

export default function RunsPage() {
  const [rawFilteredStatus, setFilteredStatus, removeFilteredStatus] =
    useStringArraySearchParam('filterStatus');

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

  const environment = useEnvironment();
  const [{ data, fetching: fetchingRuns }, refetch] = useQuery({
    query: GetRunsDocument,
    variables: {
      environmentID: environment.id,
      startTime: '2024-04-19T11:26:03.203Z',
      status: filteredStatus.length > 0 ? filteredStatus : null,
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
        <StatusFilter selectedStatuses={filteredStatus} onStatusesChange={handleStatusesChange} />
        {/* TODO: wire button */}
        <Button label="Refresh" appearance="text" btnAction={refetch} icon={<RiLoopLeftLine />} />
      </div>
      {/* @ts-expect-error */}
      <RunsTable data={runs} isLoading={fetchingRuns} />
    </main>
  );
}
