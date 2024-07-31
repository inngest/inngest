'use client';

import { Link } from '@inngest/components/Link/Link';

import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
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
  const { value: newIANav } = useBooleanFlag('new-ia-nav');
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
        {newIANav && (
          <div className="relative mb-8 flex flex-row items-center justify-between">
            <div className="flex flex-col ">
              <div className="text-basis text-2xl leading-tight">{appRes.data.name}</div>
            </div>
            <Link
              internalNavigation={true}
              href={`/env/${env.slug}/apps/${encodeURIComponent(externalID)}/syncs`}
            >
              See all synks
            </Link>
          </div>
        )}
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
