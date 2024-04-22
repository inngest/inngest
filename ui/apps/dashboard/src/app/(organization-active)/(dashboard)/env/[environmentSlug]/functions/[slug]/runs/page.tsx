'use client';

import { Button } from '@inngest/components/Button';
import { RiLoopLeftLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { FunctionRunStatus } from '@/gql/graphql';
import { useStringArraySearchParam } from '@/utils/useSearchParam';
import StatusFilter from '../logs/StatusFilter';
import RunsTable from './RunsTable';

const GetRunsDocument = graphql(`
  query GetRuns($environmentID: ID!, $startTime: Time!) {
    environment: workspace(id: $environmentID) {
      runs(filter: { from: $startTime }, orderBy: [{ field: QUEUED_AT, direction: ASC }]) {
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
  const [filteredStatus, setFilteredStatus, removeFilteredStatus] =
    useStringArraySearchParam('filterStatus');

  function handleStatusesChange(statuses: FunctionRunStatus[]) {
    if (statuses.length > 0) {
      setFilteredStatus(statuses);
    } else {
      removeFilteredStatus();
    }
  }

  const environment = useEnvironment();
  const [{ data, fetching: fetchingRuns }] = useQuery({
    query: GetRunsDocument,
    variables: {
      environmentID: environment.id,
      startTime: '2024-04-19T11:26:03.203Z',
      // filter: filtering,
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
      <div className="flex justify-between gap-2 bg-slate-50 px-8 py-2">
        <StatusFilter
          selectedStatuses={filteredStatus ? (filteredStatus as FunctionRunStatus[]) : []}
          onStatusesChange={handleStatusesChange}
        />
        {/* TODO: wire button */}
        <Button label="Refresh" appearance="text" icon={<RiLoopLeftLine />} />
      </div>
      {/* @ts-ignore */}
      <RunsTable data={runs} isLoading={fetchingRuns} />
    </main>
  );
}
