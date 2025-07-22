'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/Button';
import { EventsActionMenu } from '@inngest/components/Events/EventsActionMenu';
import { EventsTable } from '@inngest/components/Events/EventsTable';
import { useReplayModal } from '@inngest/components/Events/useReplayModal';
import { Header } from '@inngest/components/Header/Header';
import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';

import SendEventButton from '@/components/Event/SendEventButton';
import SendEventModal from '@/components/Event/SendEventModal';
import { EventInfo } from '@/components/Events/EventInfo';
import { ExpandedRowActions } from '@/components/Events/ExpandedRowActions';
import { useEventDetails, useEventPayload, useEvents } from '@/components/Events/useEvents';

const pollInterval = 400;

export default function EventsPage({
  eventTypeNames,
  showHeader = true,
}: {
  eventTypeNames?: string[];
  showHeader?: boolean;
}) {
  const router = useRouter();
  const [autoRefresh, setAutoRefresh] = useState(true);
  const { isModalVisible, selectedEvent, openModal, closeModal } = useReplayModal();

  const getEvents = useEvents();
  const getEventDetails = useEventDetails();
  const getEventPayload = useEventPayload();

  return (
    <>
      {showHeader && (
        <Header
          breadcrumb={[{ text: 'Events' }]}
          infoIcon={<EventInfo />}
          action={
            <div className="flex items-center gap-1.5">
              <SendEventButton
                label="Send event"
                data={JSON.stringify({
                  name: '',
                  data: {},
                  user: {},
                })}
              />
              <EventsActionMenu
                setAutoRefresh={() => setAutoRefresh((v) => !v)}
                autoRefresh={autoRefresh}
                intervalSeconds={pollInterval / 1000}
              />
            </div>
          }
        />
      )}
      <EventsTable
        getEvents={getEvents}
        getEventDetails={getEventDetails}
        getEventPayload={getEventPayload}
        eventNames={eventTypeNames}
        singleEventTypePage={false}
        features={{
          history: Number.MAX_SAFE_INTEGER,
        }}
        pollInterval={pollInterval}
        autoRefresh={autoRefresh}
        isDev
        emptyActions={
          <>
            <Button
              appearance="outlined"
              label="Refresh"
              onClick={() => router.refresh()}
              icon={<RiRefreshLine />}
              iconSide="left"
            />
            <Button
              label="Go to docs"
              href="https://www.inngest.com/docs/events"
              target="_blank"
              icon={<RiExternalLinkLine />}
              iconSide="left"
            />
          </>
        }
        expandedRowActions={({ eventName, payload }) => {
          return (
            <ExpandedRowActions
              eventName={eventName}
              payload={payload}
              onReplay={() => {
                if (!eventName || !payload) {
                  return;
                }
                openModal(eventName, payload);
              }}
            />
          );
        }}
      />
      {selectedEvent && (
        <SendEventModal isOpen={isModalVisible} onClose={closeModal} data={selectedEvent.data} />
      )}
    </>
  );
}
