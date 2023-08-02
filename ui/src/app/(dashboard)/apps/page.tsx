'use client';

import { useMemo } from 'react';

import AddAppButton from '@/components/App/AddAppButton';
import AppCard from '@/components/App/AppCard';
import { IconSpinner, IconWindow } from '@/icons';
import { useGetAppsQuery } from '@/store/generated';

export default function AppList() {
  const { data } = useGetAppsQuery(undefined, { pollingInterval: 1500 });
  const apps = data?.apps || [];

  const connectedApps = apps.filter((app) => app.connected === true);
  const numberOfConnectedApps = connectedApps.length;

  const memoizedAppCards = useMemo(() => {
    return apps.map((app) => {
      return <AppCard key={app?.id} app={app} />;
    });
  }, [apps]);

  return (
    <div className="px-10 py-6 h-full flex flex-col overflow-y-scroll">
      <header className="mb-8">
        <h1 className="text-lg text-slate-50">Connected Apps</h1>
        <p className="my-4">
          This is a list of all apps. We auto-detect apps that you have defined in specific ports.
        </p>
        <div className="flex items-center gap-5">
          <AddAppButton />
          <p className="text-sky-400 flex items-center gap-2">
            <IconSpinner className="fill-sky-400 text-slate-800" />
            Auto-detecting Apps
          </p>
        </div>
      </header>
      <div className="flex items-center gap-3 mb-4">
        <IconWindow className="h-5 w-5" />
        <p className="text-slate-200">
          {numberOfConnectedApps} / {apps.length} Apps Connected
        </p>
      </div>
      <div className="grid md:grid-cols-2 grid-cols-1 gap-6 min-h-max">{memoizedAppCards}</div>
    </div>
  );
}
