import { EventsHeader } from '@/components/Events/EventsHeader';
import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { EventSearch } from './EventSearch';

export default async function Page({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  return (
    <ServerFeatureFlag flag="event-search">
      <EventsHeader envSlug={envSlug} eventSearch={true} sendEvents={false} />
      <div className="bg-canvasBase flex h-full w-full flex-col">
        <EventSearch />
      </div>
    </ServerFeatureFlag>
  );
}
