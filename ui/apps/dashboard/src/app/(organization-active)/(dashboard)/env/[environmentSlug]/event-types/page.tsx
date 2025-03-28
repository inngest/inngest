'use client';

import { useMemo } from 'react';
import { EventTypesTable } from '@inngest/components/EventTypes/EventTypesTable';
import { Header } from '@inngest/components/Header/Header';

import { EventInfo } from '@/components/Events/EventInfo';
import SendEventButton from '@/components/Events/SendEventButton';
import { pathCreator } from '@/utils/urls';
import { fakeGetEventTypes } from './fakePromise';

export default function EventTypesPage({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      function: (params: { functionSlug: string }) =>
        pathCreator.function({ envSlug: envSlug, functionSlug: params.functionSlug }),
      eventType: (params: { eventName: string }) =>
        pathCreator.eventType({ envSlug: envSlug, eventName: params.eventName }),
    };
  }, [envSlug]);

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Event Types' }]}
        infoIcon={<EventInfo />}
        action={<SendEventButton />}
      />
      <EventTypesTable pathCreator={internalPathCreator} getEventTypes={fakeGetEventTypes} />
    </>
  );
}
