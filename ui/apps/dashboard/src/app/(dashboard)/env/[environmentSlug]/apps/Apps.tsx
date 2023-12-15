'use client';

import { AppCard } from './AppCard';
import { useApps } from './useApps';

type Props = {
  envID: string;
};

export function Apps({ envID }: Props) {
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
          return <AppCard className="mb-4" key={app.id} app={app} />;
        })}
      </div>
    </div>
  );
}
