'use client';

import { useState } from 'react';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { EventLogsPage } from './EventLogsPage';

type EventLogsProps = {
  eventName: string;
};

export default function EventLogs({ eventName }: EventLogsProps) {
  const [cursors, setCursors] = useState(['']);
  const environment = useEnvironment();
  const pathPrefix = `/env/${environment.slug}/events/${encodeURIComponent(eventName)}/logs`;

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
