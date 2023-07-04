'use client';

import { BlankSlate } from '@/components/Blank';
import { EventStream } from '@/components/Event/Stream';
import { TimelineScrollContainer } from '@/components/Timeline/TimelineScrollContainer';
import { useGetEventsStreamQuery } from '@/store/generated';

type LayoutProps = {
  children: React.ReactNode;
};

export default function Layout({ children }: LayoutProps) {
  const { hasEvents } = useGetEventsStreamQuery(undefined, {
    selectFromResult: (result) => ({
      ...result,
      hasEvents: Boolean(result.data?.events?.length || 0),
    }),
  });

  return (
    <div className="flex h-full">
      <TimelineScrollContainer>
        <EventStream />
      </TimelineScrollContainer>

      {hasEvents ? (
        children
      ) : (
        <BlankSlate
          title="Inngest hasn't received any events"
          subtitle="Read our documentation to learn how to send events to Inngest."
          imageUrl="/images/no-events.png"
          button={{
            text: 'Sending Events',
            onClick: () => {},
          }}
        />
      )}
    </div>
  );
}
