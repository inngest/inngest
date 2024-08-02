import { IconEvent } from '@inngest/components/icons/Event';

import { EventInfo } from '@/components/Events/EventInfo';
import { EventList } from '@/components/Events/EventList';
import SendEventButton from '@/components/Events/SendEventButton';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { Header } from '@/components/Header/Header';
import OldHeader from '@/components/Header/old/Header';

export default async function EventsPage({
  params: { environmentSlug },
}: {
  params: { environmentSlug: string };
}) {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return (
    <>
      {newIANav ? (
        <Header
          breadcrumb={[{ text: 'Events', href: `/env/${environmentSlug}/events` }]}
          icon={<EventInfo />}
          action={<SendEventButton />}
        />
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
