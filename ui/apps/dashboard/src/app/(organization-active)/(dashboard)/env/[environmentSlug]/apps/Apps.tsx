'use client';

import AppCards from '@/components/Apps/AppCards';
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

  return (
    <div className="flex items-center justify-center">
      <div className="w-full max-w-[1200px]">
        {!hasApps && !isArchived && <EmptyActiveCard envSlug={env.slug} />}
        {!hasApps && isArchived && <EmptyArchivedCard />}
        {hasApps && <AppCards apps={apps} envSlug={env.slug} />}
        {latestUnattachedSyncTime && !isArchived && (
          <UnattachedSyncsCard envSlug={env.slug} latestSyncTime={latestUnattachedSyncTime} />
        )}
      </div>
    </div>
  );
}
