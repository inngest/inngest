import EventsPage from "@/components/Events/EventsPage";
import { createFileRoute } from "@tanstack/react-router";

import EventsFeedback from "@/components/Surveys/EventsFeedback";

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/events/",
)({
  component: EventsComponent,
});

function EventsComponent() {
  const { envSlug } = Route.useParams();

  return (
    <>
      <EventsPage environmentSlug={envSlug} showHeader />
      <EventsFeedback />
    </>
  );
}
