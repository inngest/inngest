import { ClientOnly, createFileRoute, useRouter } from '@tanstack/react-router';
import { useMemo } from 'react';

import { Button } from '@inngest/components/Button';
import { EventTypesTable } from '@inngest/components/EventTypes/EventTypesTable';
import { Header } from '@inngest/components/Header/Header';
import { RefreshButton } from '@inngest/components/Refresh/RefreshButton';
import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';

import { ActionsMenu } from '@/components/EventTypes/ActionsMenu';
import { EventTypesInfo } from '@/components/EventTypes/EventTypesInfo';
import {
  useEventTypeVolume,
  useEventTypes,
} from '@/components/EventTypes/useEventTypes';
import SendEventButton from '@/components/Events/SendEventButton';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/event-types/')({
  component: EventTypesComponent,
});

export default function EventTypesComponent() {
  const { envSlug } = Route.useParams();
  const router = useRouter();

  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      function: (params: { functionSlug: string }) =>
        pathCreator.function({
          envSlug: envSlug,
          functionSlug: params.functionSlug,
        }),
      eventType: (params: { eventName: string }) =>
        pathCreator.eventType({
          envSlug: envSlug,
          eventName: params.eventName,
        }),
    };
  }, [envSlug]);
  const getEventTypes = useEventTypes();
  const getEventTypeVolume = useEventTypeVolume();

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Event Types' }]}
        infoIcon={<EventTypesInfo />}
        action={
          <div className="flex items-center gap-1.5">
            <RefreshButton />
            <SendEventButton />
          </div>
        }
      />
      <ClientOnly>
        <EventTypesTable
          key={envSlug}
          pathCreator={internalPathCreator}
          getEventTypes={getEventTypes}
          getEventTypeVolume={getEventTypeVolume}
          eventTypeActions={(props) => <ActionsMenu {...props} />}
          emptyActions={
            <>
              <Button
                appearance="outlined"
                label="Refresh"
                onClick={() => router.invalidate()}
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
        />
      </ClientOnly>
    </>
  );
}
