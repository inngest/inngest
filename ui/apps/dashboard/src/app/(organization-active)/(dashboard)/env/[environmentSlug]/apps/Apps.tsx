'use client';

import { EmptyActiveCard, EmptyArchivedCard } from '@/components/Apps/EmptyAppsCard';
import { UnattachedSyncsCard } from '@/components/Apps/UnattachedSyncsCard';
import { useEnvironment } from '@/components/Environments/environment-context';
import { AppCard, SkeletonCard } from './AppCard';
import { useApps } from './useApps';

type Props = {
  isArchived?: boolean;
};

export function Apps({ isArchived = false }: Props) {
  const env = useEnvironment();

  const res = useApps({ envID: env.id, isArchived });
  if (res.error) {
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return (
      <div className="mb-4 flex items-center justify-center">
        <div className="w-full max-w-[1200px]">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  const { apps, latestUnattachedSyncTime } = res.data;
  const hasApps = apps.length > 0;
  // Sort apps by latest sync time
  const sortedApps = apps.sort((a, b) => {
    return (
      (b.latestSync ? new Date(b.latestSync.lastSyncedAt).getTime() : 0) -
      (a.latestSync ? new Date(a.latestSync.lastSyncedAt).getTime() : 0)
    );
  });

  return (
    <div className="flex items-center justify-center">
      <div className="w-full max-w-[1200px]">
        {!hasApps && !isArchived && <EmptyActiveCard envSlug={env.slug} />}
        {!hasApps && isArchived && <EmptyArchivedCard />}
        {sortedApps.map((app) => {
          return (
            <AppCard
              app={app}
              className="mb-4"
              envSlug={env.slug}
              key={app.id}
              isArchived={isArchived}
            />
          );
        })}

        {latestUnattachedSyncTime && !isArchived && (
          <UnattachedSyncsCard envSlug={env.slug} latestSyncTime={latestUnattachedSyncTime} />
        )}
      </div>
    </div>
  );
}
