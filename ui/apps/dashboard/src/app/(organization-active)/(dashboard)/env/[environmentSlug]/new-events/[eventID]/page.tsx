'use client';

import { useCallback, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button/Button';
import { EventDetails } from '@inngest/components/Events/EventDetails';
import { RiArrowRightUpLine } from '@remixicon/react';

import { SendEventModal } from '@/components/Events/SendEventModal';
import { useEventDetails, useEventPayload, useEventRuns } from '@/components/Events/useEvents';
import { pathCreator } from '@/utils/urls';

type Props = {
  params: {
    eventID: string;
    environmentSlug: string;
  };
};

export default function Page({ params }: Props) {
  const eventID = decodeURIComponent(params.eventID);
  const envSlug = params.environmentSlug;

  const [isModalVisible, setIsModalVisible] = useState(false);
  const [selectedEvent, setSelectedEvent] = useState<{ name: string; data: string } | null>(null);

  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      eventType: (params: { eventName: string }) =>
        pathCreator.eventType({ envSlug: envSlug, eventName: params.eventName }),
      runPopout: (params: { runID: string }) =>
        pathCreator.runPopout({ envSlug: envSlug, runID: params.runID }),
      eventPopout: (params: { eventID: string }) =>
        pathCreator.eventPopout({ envSlug: envSlug, eventID: params.eventID }),
    };
  }, [envSlug]);

  const getEventDetails = useEventDetails();
  const getEventPayload = useEventPayload();
  const getEventRuns = useEventRuns();

  const openModal = useCallback((eventName: string, payload: string) => {
    try {
      const parsedData = JSON.stringify(JSON.parse(payload).data);
      setSelectedEvent({ name: eventName, data: parsedData });
      setIsModalVisible(true);
    } catch (error) {
      console.error('Failed to parse event payload:', error);
    }
  }, []);

  return (
    <>
      <EventDetails
        pathCreator={internalPathCreator}
        eventID={eventID}
        standalone
        getEventDetails={getEventDetails}
        getEventPayload={getEventPayload}
        getEventRuns={getEventRuns}
        expandedRowActions={({ eventName, payload }) => {
          const isInternalEvent = eventName?.startsWith('inngest/');
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
