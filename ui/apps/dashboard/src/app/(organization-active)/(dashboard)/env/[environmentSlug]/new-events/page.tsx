'use client';

import { useCallback, useMemo, useState } from 'react';
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
import { pathCreator } from '@/utils/urls';
import { useAccountFeatures } from '@/utils/useAccountFeatures';

export default function EventsPage({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  const router = useRouter();
  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      eventType: (params: { eventName: string }) =>
        pathCreator.eventType({ envSlug: envSlug, eventName: params.eventName }),
      runPopout: (params: { runID: string }) =>
        pathCreator.runPopout({ envSlug: envSlug, runID: params.runID }),
    };
  }, [envSlug]);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [selectedEvent, setSelectedEvent] = useState<{ name: string; data: string } | null>(null);

  const openModal = useCallback((eventName: string, payload: string) => {
    try {
      const parsedPayload = JSON.parse(payload);
      const data = Array.isArray(parsedPayload)
        ? parsedPayload.map((item) => item.data)
        : parsedPayload.data;

      const parsedData = JSON.stringify(data);
      setSelectedEvent({ name: eventName, data: parsedData });
      setIsModalVisible(true);
    } catch (error) {
      console.error('Failed to parse event payload:', error);
    }
  }, []);

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
          const isInternalEvent = Boolean(eventName.startsWith('inngest/'));
          return (
            <>
              <div className="flex items-center gap-2">
                <Button
                  label="Go to event type page"
                  href={pathCreator.eventType({ envSlug: envSlug, eventName: eventName })}
                  appearance="ghost"
                  size="small"
                  icon={<RiArrowRightUpLine />}
                  iconSide="left"
                />
                <Button
                  label="Replay event"
                  onClick={() => payload && openModal(eventName, payload)}
                  appearance="outlined"
                  size="small"
                  disabled={isInternalEvent || !payload}
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
          onClose={() => {
            setIsModalVisible(false);
            setSelectedEvent(null);
          }}
          initialData={selectedEvent.data}
        />
      )}
    </>
  );
}
