import { createFileRoute } from '@tanstack/react-router';
import EventsPage from '@/components/Events/EventsPage';

export const Route = createFileRoute('/_dashboard/events/')({
  component: EventsComponent,
});

function EventsComponent({}) {
  return <EventsPage showHeader />;
}
