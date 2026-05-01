import { type ActiveBannersQuery } from '@/gql/graphql';
import { ActiveBannerItem } from './ActiveBannerItem';

type Banner = ActiveBannersQuery['account']['activeBanners'][number];

export function ActiveBannersView({ banners }: { banners: Banner[] }) {
  return (
    <>
      {banners.map((banner) => (
        <ActiveBannerItem key={banner.id} banner={banner} />
      ))}
    </>
  );
}
