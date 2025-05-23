import { Header } from '@inngest/components/Header/Header';
import { Pill } from '@inngest/components/Pill';

import { EventTypesInfo } from '@/components/EventTypes/EventTypesInfo';
import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { pathCreator } from '@/utils/urls';
import { EventSearch } from './EventSearch';

export default async function Page({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  return (
    <ServerFeatureFlag flag="event-search">
      <Header
        breadcrumb={[{ text: 'Event Types' }]}
        infoIcon={<EventTypesInfo />}
        tabs={[
          {
            children: 'All event types',
            href: pathCreator.eventTypes({ envSlug: envSlug }),
          },
          {
            children: (
              <div className="flex flex-row items-center gap-1">
                <div>Event Search</div>
                <Pill appearance="outlined" kind="warning">
                  Experimental
                </Pill>
              </div>
            ),
            href: `/env/${envSlug}/event-search`,
          },
        ]}
      />
      <div className="bg-canvasBase flex h-full w-full flex-col">
        <EventSearch />
      </div>
    </ServerFeatureFlag>
  );
}
