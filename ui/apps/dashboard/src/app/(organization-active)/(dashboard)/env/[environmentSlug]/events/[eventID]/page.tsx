'use client';

import { EventDetails } from '@inngest/components/Events/EventDetails';
import { useReplayModal } from '@inngest/components/Events/useReplayModal';

import { ExpandedRowActions } from '@/components/Events/ExpandedRowActions';
import { SendEventModal } from '@/components/Events/SendEventModal';
import { useEventDetails, useEventPayload, useEventRuns } from '@/components/Events/useEvents';

type Props = {
  params: {
    eventID: string;
    environmentSlug: string;
  };
};

export default function Page({ params }: Props) {
  const eventID = decodeURIComponent(params.eventID);
  const envSlug = params.environmentSlug;

  const { isModalVisible, selectedEvent, openModal, closeModal } = useReplayModal();

  const getEventDetails = useEventDetails();
  const getEventPayload = useEventPayload();
  const getEventRuns = useEventRuns();

  return (
    <>
      <EventDetails
        eventID={eventID}
        standalone
        getEventDetails={getEventDetails}
        getEventPayload={getEventPayload}
        getEventRuns={getEventRuns}
        expandedRowActions={({ eventName, payload }) => (
          <ExpandedRowActions
            eventName={eventName}
            payload={payload}
            onReplay={openModal}
            envSlug={envSlug}
          />
        )}
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
