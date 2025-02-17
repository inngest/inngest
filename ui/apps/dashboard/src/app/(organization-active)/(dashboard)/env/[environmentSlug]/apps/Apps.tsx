'use client';

import { SkeletonCard } from '@inngest/components/Apps/AppCard';

import AppCards from '@/components/Apps/AppCards';
import AppFAQ from '@/components/Apps/AppFAQ';
import { EmptyActiveCard, EmptyArchivedCard } from '@/components/Apps/EmptyAppsCard';
import { UnattachedSyncsCard } from '@/components/Apps/UnattachedSyncsCard';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useApps } from './useApps';
import { useLatestUnattachedSync } from './useUnattachedSyncs';

type Props = {
  isArchived?: boolean;
};

export function Apps({ isArchived = false }: Props) {
  const env = useEnvironment();
  const unattachedSyncRes = useLatestUnattachedSync({ envID: env.id });
  if (unattachedSyncRes.error) {
    // Swallow error because we don't want to crash the page.
    console.error(unattachedSyncRes.error);
  }

  const appsRes = useApps({ envID: env.id, isArchived });
  if (appsRes.error) {
    throw appsRes.error;
  }
  if (appsRes.isLoading && !appsRes.data) {
    return (
      <div className="mb-4 flex items-center justify-center">
        <div className="w-full">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  const apps = appsRes.data;
  const hasApps = apps.length > 0;

  return (
    <div className="flex items-center justify-center">
      <div className="w-full">
        {!hasApps && !unattachedSyncRes.data && !isArchived && (
          <>
            <EmptyActiveCard envSlug={env.slug} />
            <AppFAQ />
          </>
        )}
        {!hasApps && isArchived && <EmptyArchivedCard />}
        {hasApps && <AppCards apps={apps} envSlug={env.slug} />}
        {unattachedSyncRes.data && !isArchived && (
          <>
            <UnattachedSyncsCard envSlug={env.slug} latestSyncTime={unattachedSyncRes.data} />
            {!hasApps && <AppFAQ />}
          </>
        )}
      </div>
    </div>
  );
}
