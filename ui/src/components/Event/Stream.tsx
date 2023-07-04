'use client';

import { usePathname } from 'next/navigation';
import { EventStatus, useGetEventsStreamQuery } from '../../store/generated';
import TimelineFeedContent from '../Timeline/TimelineFeedContent';
import TimelineRow from '../Timeline/TimelineRow';

export const EventStream = () => {
  const events = useGetEventsStreamQuery(undefined, { pollingInterval: 1500 });
  const pathname = usePathname();

  return (
    <>
      {events?.data?.events?.map((event, i, list) => (
        <TimelineRow
          key={event.id}
          status={event.status || EventStatus.Completed}
          iconOffset={30}
          topLine={i !== 0}
          bottomLine={i < list.length - 1}
        >
          <TimelineFeedContent
            date={event.createdAt}
            status={event.status || EventStatus.Completed}
            badge={event.totalRuns || 0}
            name={event.name || 'Unknown'}
            active={pathname.includes(event.id)}
            href={`/feed/events/${event.id}`}
          />
        </TimelineRow>
      ))}
    </>
  );
};
