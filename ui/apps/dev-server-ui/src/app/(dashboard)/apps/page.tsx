'use client';

import { useMemo } from 'react';
import { Link } from '@inngest/components/Link';

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
    <div className="flex h-full flex-col overflow-y-scroll px-10 py-6">
      <header className="mb-8">
        <h1 className="text-lg text-slate-50">Connected Apps</h1>
        <p className="my-4 flex gap-1">
          This is a list of all apps. We auto-detect apps that you have defined in{' '}
          <Link href="https://www.inngest.com/docs/local-development#connecting-apps-to-the-dev-server">
            specific ports.
          </Link>
        </p>
        <div className="flex items-center gap-5">
          <AddAppButton />
          <p className="flex items-center gap-2 text-sky-400">
            <IconSpinner />
            Auto-detecting Apps
          </p>
        </div>
      </header>
      <div className="mb-4 flex items-center gap-3">
        <IconWindow className="h-5 w-5" />
        <p className="text-slate-200">
          {numberOfConnectedApps} / {apps.length} Apps Connected
        </p>
      </div>
      <div className="grid min-h-max grid-cols-1 gap-6 md:grid-cols-2">{memoizedAppCards}</div>
    </div>
  );
}
