import { EventsHeader } from '@/components/Events/EventsHeader';
import { ServerFeatureFlag, getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { EventSearch } from './EventSearch';

export default async function Page({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  const newIANav = await getBooleanFlag('new-ia-nav');
  return (
    <ServerFeatureFlag flag="event-search">
      {newIANav && <EventsHeader envSlug={envSlug} eventSearch={true} sendEvents={false} />}
      <EventSearch />
    </ServerFeatureFlag>
  );
}
