'use client';

import { useState } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { Pill } from '@inngest/components/Pill/Pill';

import { ActionsMenu } from '@/components/Events/ActionsMenu';
import ArchiveEventModal from '@/components/Events/ArchiveEventModal';
import SendEventButton from '@/components/Events/SendEventButton';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { pathCreator } from '@/utils/urls';

type EventLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    eventTypeName: string;
  };
};

export default function EventLayout({
  children,
  params: { environmentSlug: envSlug, eventTypeName: eventSlug },
}: EventLayoutProps) {
  const eventsPath = `/env/${envSlug}/event-types/${eventSlug}/events`;
  const eventName = decodeURIComponent(eventSlug);
  const [showArchive, setShowArchive] = useState(false);

  const isNewEventsEnabled = useBooleanFlag('events-pages');

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Event types', href: pathCreator.eventTypes({ envSlug }) },
          { text: eventName, href: pathCreator.eventType({ envSlug, eventName }) },
        ]}
        tabs={[
          {
            href: pathCreator.eventType({ envSlug, eventName }),
            children: 'Dashboard',
            exactRouteMatch: true,
          },
          {
            href: eventsPath,
            children: 'Events',
          },
          ...(isNewEventsEnabled.isReady && isNewEventsEnabled.value
            ? [
                {
                  children: (
                    <div className="m-0 flex flex-row items-center justify-start space-x-1 p-0">
                      <div>Events</div>
                      <Pill kind="primary">Beta</Pill>
                    </div>
                  ),
                  href: pathCreator.eventTypeEvents({ envSlug, eventName }),
                },
              ]
            : []),
        ]}
        action={
          <div className="flex flex-row items-center justify-end gap-x-1">
            <ActionsMenu archive={() => setShowArchive(true)} />
            <SendEventButton eventName={eventName} />
          </div>
        }
      />

      <ArchiveEventModal
        eventName={eventName}
        isOpen={showArchive}
        onClose={() => {
          setShowArchive(false);
        }}
      />

      {children}
    </>
  );
}
