'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppCard } from './AppCard';
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
  if (res.isLoading) {
    return null;
  }

  return (
    <div className="mt-4 flex items-center justify-center">
      <div className="w-full max-w-[1200px]">
        {res.data.map((app) => {
          return <AppCard app={app} className="mb-4" envSlug={env.slug} key={app.id} />;
        })}
      </div>
    </div>
  );
}
