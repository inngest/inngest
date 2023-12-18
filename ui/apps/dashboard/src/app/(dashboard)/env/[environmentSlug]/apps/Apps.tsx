'use client';

import { AppCard } from './AppCard';
import { useApps } from './useApps';

type Props = {
  envID: string;
  envSlug: string;
};

export function Apps({ envID, envSlug }: Props) {
  const res = useApps(envID);
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
          return <AppCard app={app} className="mb-4" envSlug={envSlug} key={app.id} />;
        })}
      </div>
    </div>
  );
}
