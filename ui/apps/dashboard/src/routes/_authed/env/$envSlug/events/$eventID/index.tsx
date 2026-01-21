import { useReplayModal } from '@inngest/components/Events/useReplayModal';
import { createFileRoute } from '@tanstack/react-router';
import { lazy } from 'react';

import { ExpandedRowActions } from '@/components/Events/ExpandedRowActions';
import { SendEventModal } from '@/components/Events/SendEventModal';
import {
  useEventDetails,
  useEventPayload,
  useEventRuns,
} from '@/components/Events/useEvents';

const EventDetails = lazy(() =>
  import('@inngest/components/Events/EventDetails').then((mod) => ({
    default: mod.EventDetails,
  })),
);

export const Route = createFileRoute('/_authed/env/$envSlug/events/$eventID/')({
  component: EventDetailsPage,
});

function EventDetailsPage() {
  const { eventID, envSlug } = Route.useParams();

  const { isModalVisible, selectedEvent, openModal, closeModal } =
    useReplayModal();

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
