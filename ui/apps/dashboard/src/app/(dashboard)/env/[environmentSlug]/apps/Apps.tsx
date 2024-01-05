'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import PlusIcon from '@heroicons/react/20/solid/PlusIcon';
import { Button } from '@inngest/components/Button';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppCard } from './AppCard';
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
    return null;
  }

  return (
    <div className="mt-4 flex items-center justify-center">
      <div className="w-full max-w-[1200px]">
        {res.data.map((app) => {
          return <AppCard app={app} className="mb-4" envSlug={env.slug} key={app.id} />;
        })}
        {!isArchived && (
          <Button
            className="mx-auto"
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
