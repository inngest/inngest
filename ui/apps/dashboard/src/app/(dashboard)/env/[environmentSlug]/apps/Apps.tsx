'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppCard } from './AppCard';
import { useApps } from './useApps';

export function Apps() {
  const env = useEnvironment();

  const res = useApps(env.id);
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
