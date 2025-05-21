'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/Button';
import { EventsTable } from '@inngest/components/Events/EventsTable';
import { InternalEventsToggle } from '@inngest/components/Events/InternalEventsToggle';
import { Header } from '@inngest/components/Header/Header';
import { RefreshButton } from '@inngest/components/Refresh/RefreshButton';
import { RiArrowRightUpLine, RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';

import { useAllEventTypes } from '@/components/EventTypes/useEventTypes';
import { EventInfo } from '@/components/Events/EventInfo';
import SendEventButton from '@/components/Events/SendEventButton';
import { SendEventModal } from '@/components/Events/SendEventModal';
import { useEventDetails, useEventPayload, useEvents } from '@/components/Events/useEvents';
import { useReplayModal } from '@/components/Events/useReplayModal';
import { createInternalPathCreator } from '@/components/Events/utils';
import { pathCreator } from '@/utils/urls';
import { useAccountFeatures } from '@/utils/useAccountFeatures';

export default function EventsPage({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  const router = useRouter();
  const internalPathCreator = useMemo(() => createInternalPathCreator(envSlug), [envSlug]);
  const { isModalVisible, selectedEvent, openModal, closeModal } = useReplayModal();

  const getEvents = useEvents();
  const getEventDetails = useEventDetails();
  const getEventPayload = useEventPayload();
  const getEventTypes = useAllEventTypes();
  const features = useAccountFeatures();

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Events' }]}
        infoIcon={<EventInfo />}
        action={
          <div className="flex items-center gap-1.5">
            <RefreshButton />
            <SendEventButton />
            <InternalEventsToggle />
          </div>
        }
      />
      <EventsTable
        pathCreator={internalPathCreator}
        getEvents={getEvents}
        getEventDetails={getEventDetails}
        getEventPayload={getEventPayload}
        getEventTypes={getEventTypes}
        features={{
          history: features.data?.history ?? 7,
        }}
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
          const isInternalEvent = Boolean(eventName?.startsWith('inngest/'));
          return (
            <>
              <div className="flex items-center gap-2">
                <Button
                  label="Go to event type page"
                  href={
                    eventName ? pathCreator.eventType({ envSlug: envSlug, eventName }) : undefined
                  }
                  appearance="ghost"
                  size="small"
                  icon={<RiArrowRightUpLine />}
                  iconSide="left"
                  disabled={!eventName}
                />
                <Button
                  label="Replay event"
                  onClick={() => eventName && payload && openModal(eventName, payload)}
                  appearance="outlined"
                  size="small"
                  disabled={!eventName || isInternalEvent || !payload}
                />
              </div>
            </>
          );
        }}
      />
      {selectedEvent && (
        <SendEventModal
          isOpen={isModalVisible}
          eventName={selectedEvent.name}
          onClose={closeModal}
          initialData={selectedEvent.data}
        />
      )}
    </>
  );
}
