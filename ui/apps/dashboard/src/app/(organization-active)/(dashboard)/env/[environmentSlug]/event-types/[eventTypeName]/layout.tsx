'use client';

import { useState } from 'react';
import { Header } from '@inngest/components/Header/Header';

import { ActionsMenu } from '@/components/Events/ActionsMenu';
import ArchiveEventModal from '@/components/Events/ArchiveEventModal';
import SendEventButton from '@/components/Events/SendEventButton';

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
  const eventTypesPath = `/env/${envSlug}/event-types`;
  const eventsPath = `/env/${envSlug}/event-types/${eventSlug}/events`;
  const eventPath = `/env/${envSlug}/event-types/${eventSlug}`;
  const eventName = decodeURIComponent(eventSlug);
  const [showArchive, setShowArchive] = useState(false);

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Event types', href: eventTypesPath },
          { text: eventName, href: eventPath },
        ]}
        tabs={[
          {
            href: eventPath,
            children: 'Dashboard',
            exactRouteMatch: true,
          },
          {
            href: eventsPath,
            children: 'Events',
          },
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
