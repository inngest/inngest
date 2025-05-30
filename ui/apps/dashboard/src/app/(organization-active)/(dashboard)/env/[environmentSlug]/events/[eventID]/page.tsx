'use client';

import { useMemo } from 'react';
import { EventDetails } from '@inngest/components/Events/EventDetails';

import { ExpandedRowActions } from '@/components/Events/ExpandedRowActions';
import { SendEventModal } from '@/components/Events/SendEventModal';
import { useEventDetails, useEventPayload, useEventRuns } from '@/components/Events/useEvents';
import { useReplayModal } from '@/components/Events/useReplayModal';
import { createInternalPathCreator } from '@/components/Events/utils';

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

  const internalPathCreator = useMemo(() => createInternalPathCreator(envSlug), [envSlug]);

  const getEventDetails = useEventDetails();
  const getEventPayload = useEventPayload();
  const getEventRuns = useEventRuns();

  return (
    <>
      <EventDetails
        pathCreator={internalPathCreator}
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
