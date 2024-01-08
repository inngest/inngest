'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
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
      <div className="mt-4 flex items-center justify-center">
        <div className="w-full max-w-[1200px]">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  const hasApps = res.data.length > 0;

  return (
    <div className="mt-4 flex items-center justify-center">
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
        {res.data.map((app) => {
          return <AppCard app={app} className="mb-4" envSlug={env.slug} key={app.id} />;
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
