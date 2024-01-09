'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import ExclamationTriangleIcon from '@heroicons/react/20/solid/ExclamationTriangleIcon';
import PlusIcon from '@heroicons/react/20/solid/PlusIcon';
import { Button } from '@inngest/components/Button';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppCard, EmptyAppCard, SkeletonCard } from './AppCard';
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
  if (res.isLoading) {
    return (
      <div className="mb-4 mt-16 flex items-center justify-center">
        <div className="w-full max-w-[1200px]">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  const hasApps = res.data.length > 0;

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
                btnAction={() => router.push(`/env/${env.slug}/apps/sync-new` as Route)}
                icon={<PlusIcon />}
              />
            </div>
          </EmptyAppCard>
        )}
        {!hasApps && isArchived && (
          <div className="flex items-center justify-center gap-1.5 rounded-lg bg-slate-500 p-4 text-white">
            <ExclamationTriangleIcon className="h-5 w-5 text-slate-300" /> No Archived Apps
          </div>
        )}
        {res.data.map((app) => {
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
        {!isArchived && hasApps && (
          <Button
            className="mx-auto mt-12"
            kind="primary"
            label="Sync New App"
            btnAction={() => router.push(`/env/${env.slug}/apps/sync-new` as Route)}
            icon={<PlusIcon />}
          />
        )}
      </div>
    </div>
  );
}
