'use client';

import { useContext } from 'react';

import { EnvContext } from '@/contexts/env';
import { AppCard } from './AppCard';
import { useApps } from './useApps';

export function Apps() {
  const envCtx = useContext(EnvContext);
  const res = useApps(envCtx.id);
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
