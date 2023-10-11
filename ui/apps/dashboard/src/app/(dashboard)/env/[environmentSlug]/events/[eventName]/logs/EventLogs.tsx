'use client';

import { useState } from 'react';

import LoadingIcon from '@/icons/LoadingIcon';
import { useEnvironment } from '@/queries';
import { EventLogsPage } from './EventLogsPage';

type EventLogsProps = {
  environmentSlug: string;
  eventName: string;
};

export default function EventLogs({ environmentSlug, eventName }: EventLogsProps) {
  const [cursors, setCursors] = useState(['']);
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });

  if (isFetchingEnvironment) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  // Should be impossible.
  if (!environment) {
    return null;
  }

  const pathPrefix = `/env/${environmentSlug}/events/${encodeURIComponent(eventName)}/logs`;

  function loadNextPage(cursor: string) {
    setCursors((cursors) => {
      if (cursors.includes(cursor)) {
        return cursors;
      }
      return [...cursors, cursor];
    });

    cursors.push(cursor);
  }

  return (
    <ul role="list" className="h-full divide-y divide-slate-100">
      {cursors.map((cursor, index) => {
        return (
          <EventLogsPage
            cursor={cursor}
            environmentID={environment.id}
            eventName={eventName}
            isFirstPage={index === 0}
            isLastPage={index === cursors.length - 1}
            onLoadMore={(cursor) => {
              setCursors((cursors) => {
                // Just in case the callback is called multiple times for the
                // same cursor.
                if (cursors.includes(cursor)) {
                  return cursors;
                }

                return [...cursors, cursor];
              });
            }}
            key={cursor}
            pathPrefix={pathPrefix}
          />
        );
      })}
    </ul>
  );
}
