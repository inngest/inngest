'use client';

import { EventDetails } from '@inngest/components/Events/EventDetails';
import { useReplayModal } from '@inngest/components/Events/useReplayModal';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';

import SendEventModal from '@/components/Event/SendEventModal';
import { ExpandedRowActions } from '@/components/Events/ExpandedRowActions';
import { useEventDetails, useEventPayload, useEventRuns } from '@/components/Events/useEvents';
import { useFeatureFlags } from '@/hooks/useFeatureFlags';

export default function Page() {
  const [eventID] = useSearchParam('eventID');
  const { isModalVisible, selectedEvent, openModal, closeModal } = useReplayModal();

  const getEventDetails = useEventDetails();
  const getEventPayload = useEventPayload();
  const getEventRuns = useEventRuns();

  const { featureFlags } = useFeatureFlags();
  const isEventsEnabled = featureFlags.FEATURE_EVENTS;

  if (!eventID) {
    throw new Error('missing eventID in search params');
  }

  if (!isEventsEnabled) return null;

  return (
    <>
      <EventDetails
        eventID={eventID}
        standalone
        getEventDetails={getEventDetails}
        getEventPayload={getEventPayload}
        getEventRuns={getEventRuns}
        expandedRowActions={({ eventName, payload }) => (
          <ExpandedRowActions eventName={eventName} payload={payload} onReplay={openModal} />
        )}
      />
      {selectedEvent && (
        <SendEventModal isOpen={isModalVisible} onClose={closeModal} data={selectedEvent.data} />
      )}
    </>
  );
}
