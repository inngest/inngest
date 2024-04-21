'use client';

// import { useQuery } from 'urql';

// import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
// import { graphql } from '@/gql';
import { ArrowPathIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import { FunctionRunStatus } from '@/gql/graphql';
import { useStringArraySearchParam } from '@/utils/useSearchParam';
import StatusFilter from '../logs/StatusFilter';
import RunsTable from './RunsTable';
import { mockedRuns } from './mockedRuns';

// const GetRunsDocument = graphql(`
//   query GetRuns($environmentID: ID!) {
//     workspace(id: $environmentID) {
//       runs {
//       }
//     }
//   }
// `);

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
  //   const environment = useEnvironment();
  //   const [{ data, fetching: fetchingRuns }] = useQuery({
  //     query: GetRunsDocument,
  //     variables: {
  //       environmentID: environment.id,
  //       // filter: filtering,
  //     },
  //   });

  const runs = mockedRuns;

  return (
    <main className="min-h-0 overflow-y-auto bg-white">
      <div className="flex justify-between gap-2 bg-slate-50 px-8 py-2">
        <StatusFilter
          selectedStatuses={(filteredStatus as FunctionRunStatus[]) ?? []}
          onStatusesChange={handleStatusesChange}
        />
        {/* TODO: wire button */}
        <Button label="Refresh" appearance="text" icon={<ArrowPathIcon />} />
      </div>
      {/* @ts-ignore */}
      <RunsTable data={runs} />
    </main>
  );
}
