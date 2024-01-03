'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppInfoCard } from './AppInfoCard';
import { FunctionList } from './FunctionList';
import { GitCard } from './GitCard';
import { useApp } from './useApp';

type Props = {
  params: {
    environmentSlug: string;
    externalID: string;
  };
};

export default function Page({ params: { environmentSlug, externalID } }: Props) {
  externalID = decodeURIComponent(externalID);
  const env = useEnvironment();

  const appRes = useApp({
    envID: env.id,
    externalAppID: externalID,
  });
  if (appRes.error) {
    throw appRes.error;
  }
  if (appRes.isLoading) {
    return null;
  }

  const { syncedFunctions } = appRes.data.latestSync ?? {};

  return (
    <div className="flex items-center justify-center bg-slate-100 pt-4">
      <div className="w-full max-w-[1200px]">
        <AppInfoCard app={appRes.data} className="mb-4" />

        {appRes.data.latestSync && <GitCard className="mb-4" sync={appRes.data.latestSync} />}

        {syncedFunctions && <FunctionList envSlug={environmentSlug} functions={syncedFunctions} />}
      </div>
    </div>
  );
}
