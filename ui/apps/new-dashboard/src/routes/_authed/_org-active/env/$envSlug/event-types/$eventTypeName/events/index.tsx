import { createFileRoute } from "@tanstack/react-router";

import EventsPage from "@/components/Events/EventsPage";

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/event-types/$eventTypeName/events/",
)({
  component: EventTypeEventsPage,
});

function EventTypeEventsPage() {
  const { envSlug, eventTypeName } = Route.useParams();

  return (
    <EventsPage
      environmentSlug={envSlug}
      eventTypeNames={[eventTypeName]}
      singleEventTypePage
      showHeader={false}
    />
  );
}
