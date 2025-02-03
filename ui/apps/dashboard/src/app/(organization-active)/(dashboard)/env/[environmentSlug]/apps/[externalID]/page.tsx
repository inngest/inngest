'use client';

import { FunctionList } from '@inngest/components/Apps/FunctionList';
import { Button } from '@inngest/components/Button/Button';
import { RiListCheck } from '@remixicon/react';

import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { useEnvironment } from '@/components/Environments/environment-context';
import { SyncErrorCard } from '@/components/SyncErrorCard';
import { pathCreator } from '@/utils/urls';
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
      <div className="mx-auto my-12 flex w-full max-w-[1200px] flex-col gap-9 px-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="mb-1 text-2xl">{appRes.data.name}</h2>
            <p className="text-muted text-sm">Information about the latest successful sync.</p>
          </div>
          <Button
            appearance="outlined"
            iconSide="left"
            icon={<RiListCheck />}
            href={`/env/${env.slug}/apps/${encodeURIComponent(externalID)}/syncs`}
            label="See all syncs"
          />
        </div>

        {appRes.data.latestSync?.error && (
          <SyncErrorCard className="mb-4" error={appRes.data.latestSync.error} />
        )}

        <AppInfoCard app={appRes.data} sync={appRes.data.latestSync} />

        {appRes.data.latestSync && <AppGitCard className="mb-4" sync={appRes.data.latestSync} />}

        <div>
          <h4 className="text-subtle mb-4 text-xl">
            Function list ({appRes.data.functions.length})
          </h4>
          <FunctionList
            envSlug={environmentSlug}
            functions={appRes.data.functions}
            pathCreator={{ function: pathCreator.function, eventType: pathCreator.eventType }}
          />
        </div>
      </div>
    </div>
  );
}
