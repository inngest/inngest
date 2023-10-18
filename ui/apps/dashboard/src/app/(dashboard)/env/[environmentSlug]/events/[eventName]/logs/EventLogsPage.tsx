'use client';

import { type Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useQuery } from 'urql';

import Button from '@/components/Button';
import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import cn from '@/utils/cn';
import { relativeTime, weekDayAndUTCTime } from '@/utils/date';

const perPage = 50;

const GetEventLogsDocument = graphql(`
  query GetEventLog($environmentID: ID!, $eventName: String!, $cursor: String, $perPage: Int!) {
    environment: workspace(id: $environmentID) {
      eventType: event(name: $eventName) {
        events: recent @cursored(cursor: $cursor, perPage: $perPage) {
          id
          receivedAt
        }
      }
    }
  }
`);

type Props = {
  cursor: string;
  environmentID: string;
  eventName: string;
  isFirstPage: boolean;
  isLastPage: boolean;
  onLoadMore: (cursor: string) => void;
  pathPrefix: string;
};

export function EventLogsPage({
  cursor,
  environmentID,
  eventName,
  isFirstPage,
  isLastPage,
  onLoadMore,
  pathPrefix,
}: Props) {
  const [{ data, fetching }] = useQuery({
    query: GetEventLogsDocument,
    variables: {
      // API expects "unset cursor" to be undefined, so change empty strings to
      // undefined.
      cursor: cursor || null,

      environmentID,
      eventName,
      perPage,
    },
  });

  const pathname = usePathname();

  if (fetching) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const eventType = data?.environment.eventType;

  if (!eventType) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <h2 className="text-sm font-semibold text-gray-900">Event type not found</h2>
      </div>
    );
  }

  const events = eventType.events;
  const lastEvent = events[events.length - 1];
  if (!lastEvent) {
    // No events at all, since this is the first page and the query returned 0 events.
    if (isFirstPage) {
      return (
        <div className="flex h-full w-full items-center justify-center">
          <h2 className="text-sm font-semibold text-gray-900">No events yet</h2>
        </div>
      );
    }

    // Previous page had the last event, so this page is empty.
    return null;
  }

  // Show the "Load More" button if this is the last page and there are more
  // events to load. We'll assume there are more events to load if the page is
  // full. This is a flawed assumption since the last event could be in this
  // page, but we'll handle that scenario separately.
  const isLoadMoreVisible = isLastPage && events.length === perPage;

  return (
    <>
      {events.map((event) => {
        const eventPathname = `${pathPrefix}/${event.id}`;
        const isActive = pathname === eventPathname;

        return (
          <li key={event.id}>
            <Link
              href={eventPathname as Route}
              className={cn(
                'flex items-center gap-3 px-3 py-2.5 hover:bg-slate-100',
                isActive && 'bg-slate-100'
              )}
            >
              <div className="flex min-w-0 flex-col gap-1">
                <Time
                  className="truncate text-sm font-semibold text-slate-800"
                  format="relative"
                  value={new Date(event.receivedAt)}
                />

                <Time
                  className="runcate text-sm text-slate-400"
                  value={new Date(event.receivedAt)}
                />
              </div>
            </Link>
          </li>
        );
      })}

      {isLoadMoreVisible && (
        <div className="flex justify-center">
          <Button className="mt-4" onClick={() => onLoadMore(lastEvent.id)} variant="secondary">
            Load More
          </Button>
        </div>
      )}
    </>
  );
}
