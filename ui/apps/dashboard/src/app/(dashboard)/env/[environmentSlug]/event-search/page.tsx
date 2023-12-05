import { EventSearch } from './EventSearch';

type Props = {
  params: {
    environmentSlug: string;
  };
};

export default async function Page({ params: { environmentSlug } }: Props) {
  // TEMPORARILY DISABLED TO TEST IF LAUNCHDARKLY IS CAUSING ISSUES
  return (
    <></>
    // <ServerFeatureFlag flag="event-search">
    //   <EventSearch environmentSlug={environmentSlug} />
    // </ServerFeatureFlag>
  );
}
