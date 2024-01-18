'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import PlusIcon from '@heroicons/react/20/solid/PlusIcon';
import { Button } from '@inngest/components/Button';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { pathCreator } from '@/utils/urls';
import { AppCard, EmptyAppCard, SkeletonCard } from './AppCard';
import { UnattachedSyncsCard } from './UnattachedSyncsCard';
import { useApps } from './useApps';

type Props = {
  isArchived?: boolean;
};

export function Apps({ isArchived = false }: Props) {
  const env = useEnvironment();
  const router = useRouter();

  const res = useApps({ envID: env.id, isArchived });
  if (res.error) {
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return (
      <div className="mb-4 mt-16 flex items-center justify-center">
        <div className="w-full max-w-[1200px]">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  const { apps, latestUnattachedSyncTime } = res.data;
  const hasApps = apps.length > 0;

  return (
    <div className="mb-4 mt-16 flex items-center justify-center">
      <div className="w-full max-w-[1200px]">
        {!hasApps && !isArchived && (
          <EmptyAppCard>
            <div>
              <Button
                className="mt-4"
                kind="primary"
                label="Sync App"
                btnAction={() => router.push(pathCreator.createApp({ envSlug: env.slug }))}
                icon={<PlusIcon />}
              />
            </div>
          </EmptyAppCard>
        )}
        {!hasApps && isArchived && (
          <p className="rounded-lg bg-slate-500 p-4 text-center text-white">No archived apps</p>
        )}
        {apps.map((app) => {
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

        {!isArchived && hasApps && (
          <Button
            className="mx-auto my-12"
            kind="primary"
            label="Sync New App"
            btnAction={() => router.push(pathCreator.createApp({ envSlug: env.slug }))}
            icon={<PlusIcon />}
          />
        )}
      </div>
    </div>
  );
}
