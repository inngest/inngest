import { EventTypesTable } from '@inngest/components/EventTypes/EventTypesTable';
import { Header } from '@inngest/components/Header/Header';

import { EventInfo } from '@/components/Events/EventInfo';
import SendEventButton from '@/components/Events/SendEventButton';
import { fakeGetEventTypes } from './fakePromise';

export default async function EventTypesPage({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  console.log(envSlug);
  return (
    <>
      <Header
        breadcrumb={[{ text: 'Events' }]}
        infoIcon={<EventInfo />}
        action={<SendEventButton />}
      />
      <EventTypesTable envID="env_123" getEventTypes={fakeGetEventTypes} />
    </>
  );
}
