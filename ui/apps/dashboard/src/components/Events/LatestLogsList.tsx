import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import { IDCell, TimeCell } from '@inngest/components/Table/Cell';
import { RiKey2Fill } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const GetLatestEventLogs = graphql(`
  query GetLatestEventLogs($name: String, $environmentID: ID!) {
    events(query: { name: $name, workspaceID: $environmentID }) {
      data {
        recent(count: 5) {
          id
          receivedAt
          event
          source {
            name
          }
        }
      }
    }
  }
`);

type LatestLogsListProps = {
  environmentSlug: string;
  eventName: string;
};

export default function LatestLogsList({ environmentSlug, eventName }: LatestLogsListProps) {
  const environment = useEnvironment();
  const router = useRouter();

  const [{ data: LatestLogsResponse, fetching: fetchingLatestLogs }] = useQuery({
    query: GetLatestEventLogs,
    variables: {
      environmentID: environment.id,
      name: eventName,
    },
  });

  const list = LatestLogsResponse?.events?.data[0]?.recent;

  const orderedList = list?.sort((a: { receivedAt: string }, b: { receivedAt: string }) => {
    return new Date(b.receivedAt).getTime() - new Date(a.receivedAt).getTime();
  });

  return (
    <>
      <header className="flex items-center justify-between gap-3 py-3 pl-4 pr-2">
        <h1 className="text-basis font-medium">Latest</h1>
        <Button
          appearance="outlined"
          kind="secondary"
          href={`/env/${environmentSlug}/events/${encodeURIComponent(eventName)}/logs` as Route}
          label="View all logs"
        />
      </header>

      <main className="text-basis mx-2 min-h-0 flex-1 overflow-y-auto">
        <div className="border-subtle bg-canvasBase rounded-md border text-sm">
          <table className="divide-subtle w-full divide-y rounded-md">
            <thead className="h-full text-left">
              <tr>
                <th className="p-4 font-semibold" scope="col">
                  Received at
                </th>
                <th className="font-semibold" scope="col">
                  ID
                </th>
                <th className="font-semibold" scope="col">
                  Source
                </th>
              </tr>
            </thead>
            <tbody className="divide-subtle h-full divide-y">
              {!orderedList && fetchingLatestLogs && (
                <tr>
                  <td className="p-4 text-center" colSpan={3}>
                    Loading...
                  </td>
                </tr>
              )}
              {orderedList && orderedList.length > 0
                ? orderedList.map((e) => (
                    <tr
                      className="hover:bg-canvasSubtle/50 cursor-pointer truncate transition-all"
                      key={e.id}
                      onClick={() =>
                        router.push(
                          `/env/${environmentSlug}/events/${encodeURIComponent(eventName)}/logs/${
                            e.id
                          }`
                        )
                      }
                    >
                      <td className="flex items-center gap-6 p-4">
                        <TimeCell format="relative" date={new Date(e.receivedAt)} />
                        <TimeCell date={new Date(e.receivedAt)} />
                      </td>
                      <td>
                        <IDCell>{e.id}</IDCell>
                      </td>
                      <td>
                        <Pill appearance="outlined">
                          <RiKey2Fill className="text-basis h-4 pr-1" />
                          {e.source?.name}
                        </Pill>
                      </td>
                    </tr>
                  ))
                : !fetchingLatestLogs && (
                    <tr>
                      <td className="p-4 text-center" colSpan={3}>
                        No events were stored yet.
                      </td>
                    </tr>
                  )}
            </tbody>
          </table>
        </div>
      </main>
    </>
  );
}
