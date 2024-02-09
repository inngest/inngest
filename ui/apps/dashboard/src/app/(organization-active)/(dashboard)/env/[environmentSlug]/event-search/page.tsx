import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { EventSearch } from './EventSearch';

export default async function Page() {
  return (
    <ServerFeatureFlag flag="event-search">
      <EventSearch />
    </ServerFeatureFlag>
  );
}
