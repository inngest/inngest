import { IconEvent } from '@inngest/components/icons/Event';

import { EventList } from '@/components/Events/EventList';
import { EventsHeader } from '@/components/Events/EventsHeader';
import SendEventButton from '@/components/Events/SendEventButton';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import OldHeader from '@/components/Header/old/Header';

export default async function EventsPage({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  const newIANav = await getBooleanFlag('new-ia-nav');
  const eventSearch = await getBooleanFlag('event-search');

  return (
    <>
      {newIANav ? (
        <EventsHeader envSlug={envSlug} eventSearch={eventSearch} sendEvents={true} />
      ) : (
        <OldHeader
          title="Events"
          icon={<IconEvent className="h-4 w-4 text-white" />}
          action={<SendEventButton />}
        />
      )}
      <EventList />
    </>
  );
}
