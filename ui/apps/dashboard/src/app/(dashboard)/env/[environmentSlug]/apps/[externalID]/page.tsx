'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { FunctionList } from './FunctionList';
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
    return (
      <div className="h-full overflow-y-auto">
        <div className="mx-auto w-full max-w-[1200px] py-4">
          <AppInfoCard className="mb-4" loading />
        </div>
      </div>
    );
  }

  const { syncedFunctions } = appRes.data.latestSync ?? {};

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto w-full max-w-[1200px] py-4">
        <AppInfoCard app={appRes.data} className="mb-4" sync={appRes.data.latestSync} />

        {appRes.data.latestSync && <AppGitCard className="mb-4" sync={appRes.data.latestSync} />}

        {syncedFunctions && <FunctionList envSlug={environmentSlug} functions={syncedFunctions} />}
      </div>
    </div>
  );
}
