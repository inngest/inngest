import { useQuery } from '@tanstack/react-query';

import { activeBanners } from '@/queries/server/activeBanners';
import { type ActiveBannersQuery } from '@/gql/graphql';
import { ActiveBannersView } from './ActiveBannersView';

type Banner = ActiveBannersQuery['account']['activeBanners'][number];

export function ActiveBanners() {
  const { data } = useQuery<Banner[]>({
    queryKey: ['activeBanners'],
    queryFn: async () => {
      try {
        return await activeBanners();
      } catch (err) {
        // Banners are optional UI; log for diagnostics but degrade to
        // rendering nothing rather than crashing the layout.
        console.error('activeBanners fetch failed:', err);
        return [];
      }
    },
    // Matches the server-side match-cache TTL so we're not re-asking
    // across every navigation while a banner is steady.
    staleTime: 60_000,
    // Poll every 5 minutes so banner activation/deactivation propagates
    // even for users sitting on a single page without navigating or
    // refocusing the tab.
    refetchInterval: 5 * 60_000,
  });

  if (!data || data.length === 0) return null;
  return <ActiveBannersView banners={data} />;
}
