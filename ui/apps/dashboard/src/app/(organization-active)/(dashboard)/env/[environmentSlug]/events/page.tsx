import { EventList } from '@/components/Events/EventList';
import { EventsHeader } from '@/components/Events/EventsHeader';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';

export default async function EventsPage({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  const eventSearch = await getBooleanFlag('event-search');

  return (
    <>
      <EventsHeader envSlug={envSlug} eventSearch={eventSearch} sendEvents={true} />
      <EventList />
    </>
  );
}
