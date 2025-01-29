'use client';

import { AppCard } from '@inngest/components/Apps/AppCard';

import { EmptyActiveCard, EmptyArchivedCard } from '@/components/Apps/EmptyAppsCard';
import { UnattachedSyncsCard } from '@/components/Apps/UnattachedSyncsCard';
import { useEnvironment } from '@/components/Environments/environment-context';
import { SkeletonCard } from './AppCard';
import { useApps } from './useApps';

type Props = {
  isArchived?: boolean;
};

export function Apps({ isArchived = false }: Props) {
  const env = useEnvironment();

  const res = useApps({ envID: env.id, isArchived });
  console.log(res);
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
      (b.lastSyncedAt ? new Date(b.lastSyncedAt).getTime() : 0) -
      (a.lastSyncedAt ? new Date(a.lastSyncedAt).getTime() : 0)
    );
  });

  return (
    <div className="flex items-center justify-center">
      <div className="w-full max-w-[1200px]">
        {!hasApps && !isArchived && <EmptyActiveCard envSlug={env.slug} />}
        {!hasApps && isArchived && <EmptyArchivedCard />}
        {sortedApps.map((app) => {
          return (
            <div className="mb-6" key={app.id}>
              <AppCard
                // TO DO: make the error and warning status
                kind={isArchived ? 'default' : 'primary'}
              >
                <AppCard.Content
                  app={app}
                  // TO DO: build pills and actions
                  pill={<></>}
                  actions={<></>}
                ></AppCard.Content>
              </AppCard>
            </div>
          );
        })}

        {latestUnattachedSyncTime && !isArchived && (
          <UnattachedSyncsCard envSlug={env.slug} latestSyncTime={latestUnattachedSyncTime} />
        )}
      </div>
    </div>
  );
}
