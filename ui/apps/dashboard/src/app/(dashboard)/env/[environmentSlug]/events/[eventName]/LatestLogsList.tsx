import { type Route } from 'next';
import Link from 'next/link';
import { KeyIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { useQuery } from 'urql';

import { Pill } from '@/components/Pill/Pill';
import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';
import { defaultTime, relativeTime } from '@/utils/date';

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
  const [{ data: environment, fetching: fetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });

  const [{ data: LatestLogsResponse, fetching: fetchingLatestLogs }] = useQuery({
    query: GetLatestEventLogs,
    variables: {
      environmentID: environment?.id!,
      name: eventName,
    },
    pause: !environment?.id,
  });

  const list = LatestLogsResponse?.events?.data[0]?.recent;

  const orderedList = list?.sort((a: { receivedAt: string }, b: { receivedAt: string }) => {
    return new Date(b.receivedAt).getTime() - new Date(a.receivedAt).getTime();
  });

  return (
    <>
      <header className="flex items-center gap-3 py-3 pl-4 pr-2">
        <h1 className="font-semibold text-slate-700">Latest</h1>
        <Button
          className="ml-auto"
          appearance="outlined"
          href={`/env/${environmentSlug}/events/${encodeURIComponent(eventName)}/logs` as Route}
          label="View All Logs"
        />
      </header>

      <main className="mx-2 min-h-0 flex-1 overflow-y-auto">
        <div className="rounded-md border border-slate-200 text-sm text-slate-500">
          <table className="w-full table-fixed divide-y divide-slate-200 rounded-lg bg-white">
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
            <tbody className="h-full divide-y divide-slate-200">
              {!orderedList && (fetchingEnvironment || fetchingLatestLogs) && (
                <tr>
                  <td className="p-4 text-center" colSpan={3}>
                    Loading...
                  </td>
                </tr>
              )}
              {orderedList && orderedList?.length > 0
                ? orderedList?.map((e) => (
                    <tr className="truncate" key={e.id}>
                      <td className="p-4">
                        <Time
                          className="inline-flex min-w-[112px] pr-6 font-semibold text-slate-700"
                          format="relative"
                          value={new Date(e.receivedAt)}
                        />
                        <Time value={new Date(e.receivedAt)} />
                      </td>
                      <td className="font-mono text-xs font-medium">
                        <Link
                          href={
                            `/env/${environmentSlug}/events/${encodeURIComponent(eventName)}/logs/${
                              e.id
                            }` as Route
                          }
                          className="hover:underline"
                        >
                          {e.id}
                        </Link>
                      </td>
                      <td>
                        <Pill>
                          <KeyIcon className="h-4 pr-1 text-indigo-500" />
                          {e.source?.name}
                        </Pill>
                      </td>
                    </tr>
                  ))
                : !fetchingEnvironment &&
                  !fetchingLatestLogs && (
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
