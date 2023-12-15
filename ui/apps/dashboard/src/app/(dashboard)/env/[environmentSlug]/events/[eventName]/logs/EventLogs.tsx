'use client';

import { useContext, useState } from 'react';

import { EnvContext } from '@/contexts/env';
import { EventLogsPage } from './EventLogsPage';

type EventLogsProps = {
  eventName: string;
};

export default function EventLogs({ eventName }: EventLogsProps) {
  const [cursors, setCursors] = useState(['']);
  const env = useContext(EnvContext);
  const pathPrefix = `/env/${env.slug}/events/${encodeURIComponent(eventName)}/logs`;

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
            environmentID={env.id}
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
