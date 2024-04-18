'use client';

// import { useQuery } from 'urql';

// import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
// import { graphql } from '@/gql';
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
    <main className="bg-white">
      <div className="m-8 flex gap-2">{/* TODO: filters */}</div>
      {/* @ts-ignore */}
      <RunsTable data={runs} />
    </main>
  );
}
