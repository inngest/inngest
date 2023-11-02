import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { EventSearch } from './EventSearch';

type Props = {
  params: {
    environmentSlug: string;
  };
};

export default async function Page({ params: { environmentSlug } }: Props) {
  return (
    <ServerFeatureFlag flag="event-search">
      <EventSearch environmentSlug={environmentSlug} />
    </ServerFeatureFlag>
  );
}
