"use client";

import { useRouter } from "next/navigation";
import { Button } from "@inngest/components/Button/Button";
import { EventsActionMenu } from "@inngest/components/Events/EventsActionMenu";
import { EventsTable } from "@inngest/components/Events/EventsTable";
import { useReplayModal } from "@inngest/components/Events/useReplayModal";
import { Header } from "@inngest/components/Header/Header";
import { RefreshButton } from "@inngest/components/Refresh/RefreshButton";
import { RiExternalLinkLine, RiRefreshLine } from "@remixicon/react";

import { useAllEventTypes } from "@/components/EventTypes/useEventTypes";
import { EventInfo } from "@/components/Events/EventInfo";
import { ExpandedRowActions } from "@/components/Events/ExpandedRowActions";
import SendEventButton from "@/components/Events/SendEventButton";
import { SendEventModal } from "@/components/Events/SendEventModal";
import {
  useEventDetails,
  useEventPayload,
  useEvents,
} from "@/components/Events/useEvents";
import { useAccountFeatures } from "@/utils/useAccountFeatures";

export default function EventsPage({
  environmentSlug: envSlug,
  eventTypeNames,
  showHeader = true,
  singleEventTypePage = false,
}: {
  environmentSlug: string;
  eventTypeNames?: string[];
  showHeader?: boolean;
  singleEventTypePage?: boolean;
}) {
  const router = useRouter();
  const { isModalVisible, selectedEvent, openModal, closeModal } =
    useReplayModal();

  const getEvents = useEvents();
  const getEventDetails = useEventDetails();
  const getEventPayload = useEventPayload();
  const getEventTypes = useAllEventTypes();
  const features = useAccountFeatures();

  return (
    <>
      {showHeader && (
        <Header
          breadcrumb={[{ text: "Events" }]}
          infoIcon={<EventInfo />}
          action={
            <div className="flex items-center gap-1.5">
              <RefreshButton />
              <SendEventButton />
              <EventsActionMenu />
            </div>
          }
        />
      )}
      <EventsTable
        getEvents={getEvents}
        getEventDetails={getEventDetails}
        getEventPayload={getEventPayload}
        getEventTypes={getEventTypes}
        eventNames={eventTypeNames}
        singleEventTypePage={singleEventTypePage}
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
