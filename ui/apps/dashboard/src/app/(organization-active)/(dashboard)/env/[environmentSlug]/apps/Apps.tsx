'use client';

import { Alert } from '@inngest/components/Alert/Alert';
import { SkeletonCard } from '@inngest/components/Apps/AppCard';

import AppCards from '@/components/Apps/AppCards';
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
        <div className="w-full max-w-[1200px]">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  const apps = appsRes.data;
  const hasApps = apps.length > 0;

  return (
    <div className="flex items-center justify-center">
      <div className="w-full max-w-[1200px]">
        {!hasApps && !unattachedSyncRes.data && !isArchived && (
          <EmptyActiveCard envSlug={env.slug} />
        )}
        {!hasApps && isArchived && <EmptyArchivedCard />}
        {hasApps && <AppCards apps={apps} envSlug={env.slug} />}
        {unattachedSyncRes.data && !isArchived && (
          <>
            <UnattachedSyncsCard envSlug={env.slug} latestSyncTime={unattachedSyncRes.data} />
            {!hasApps && (
              <Alert
                className="flex items-center justify-between text-sm"
                link={
                  <Alert.Link
                    severity="info"
                    href="https://www.inngest.com/docs/apps/cloud#troubleshooting?ref=apps-unattached-sync"
                    target="_blank"
                  >
                    Go to docs
                  </Alert.Link>
                }
                severity="info"
              >
                Having trouble syncing an app? Check our documentation.
              </Alert>
            )}
          </>
        )}
      </div>
    </div>
  );
}
