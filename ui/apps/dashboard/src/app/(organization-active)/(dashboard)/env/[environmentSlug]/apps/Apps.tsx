'use client';

import { Alert } from '@inngest/components/Alert/Alert';
import { SkeletonCard } from '@inngest/components/Apps/AppCard';

import AppCards from '@/components/Apps/AppCards';
import { EmptyActiveCard, EmptyArchivedCard } from '@/components/Apps/EmptyAppsCard';
import { UnattachedSyncsCard } from '@/components/Apps/UnattachedSyncsCard';
import { useEnvironment } from '@/components/Environments/environment-context';
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
        {!hasApps && !latestUnattachedSyncTime && !isArchived && (
          <EmptyActiveCard envSlug={env.slug} />
        )}
        {!hasApps && isArchived && <EmptyArchivedCard />}
        {hasApps && <AppCards apps={apps} envSlug={env.slug} envID={env.id} />}
        {latestUnattachedSyncTime && !isArchived && (
          <>
            <UnattachedSyncsCard envSlug={env.slug} latestSyncTime={latestUnattachedSyncTime} />
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
