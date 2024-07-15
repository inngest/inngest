'use client';

import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { useEnvironment } from '@/components/Environments/environment-context';
import { SyncErrorCard } from '@/components/SyncErrorCard';
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
    if (!appRes.data) {
      throw appRes.error;
    }
    console.error(appRes.error);
  }
  if (appRes.isLoading && !appRes.data) {
    return (
      <div className="h-full overflow-y-auto">
        <div className="mx-auto w-full max-w-[1200px] py-4">
          <AppInfoCard className="mb-4" loading />
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto w-full max-w-[1200px] py-4">
        {appRes.data.latestSync?.error && (
          <SyncErrorCard className="mb-4" error={appRes.data.latestSync.error} />
        )}

        <AppInfoCard app={appRes.data} className="mb-4" sync={appRes.data.latestSync} />

        {appRes.data.latestSync && <AppGitCard className="mb-4" sync={appRes.data.latestSync} />}

        <FunctionList envSlug={environmentSlug} functions={appRes.data.functions} />
      </div>
    </div>
  );
}
