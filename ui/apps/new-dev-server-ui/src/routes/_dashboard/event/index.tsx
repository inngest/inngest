import { createFileRoute } from '@tanstack/react-router'
import { EventDetails } from '@inngest/components/Events/EventDetails'
import { useReplayModal } from '@inngest/components/Events/useReplayModal'
import { useSearchParam } from '@inngest/components/hooks/useNewSearchParams'

import SendEventModal from '@/components/Event/SendEventModal'
import { ExpandedRowActions } from '@/components/Events/ExpandedRowActions'
import {
  useEventDetails,
  useEventPayload,
  useEventRuns,
} from '@/components/Events/useEvents'

export const Route = createFileRoute('/_dashboard/event/')({
  component: EventComponent,
})

function EventComponent() {
  const [eventID] = useSearchParam('eventID')
  const { isModalVisible, selectedEvent, openModal, closeModal } =
    useReplayModal()

  const getEventDetails = useEventDetails()
  const getEventPayload = useEventPayload()
  const getEventRuns = useEventRuns()

  if (!eventID) {
    throw new Error('missing eventID in search params')
  }

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
          />
        )}
      />
      {selectedEvent && (
        <SendEventModal
          isOpen={isModalVisible}
          onClose={closeModal}
          data={selectedEvent.data}
        />
      )}
    </>
  )
}
