import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppInfoCard } from '@/components/AppInfoCard';
import { useApp } from '../useApp';

type Props = {
  externalAppID: string;
};

export function Sync({ externalAppID }: Props) {
  const env = useEnvironment();

  const appRes = useApp({
    envID: env.id,
    externalAppID,
  });
  if (appRes.error) {
    throw appRes.error;
  }
  if (appRes.isLoading) {
    return null;
  }

  return <AppInfoCard app={appRes.data} sync={appRes.data.latestSync} />;
}
