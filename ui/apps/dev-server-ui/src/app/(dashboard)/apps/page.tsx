'use client';

import { useMemo } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { IconApp } from '@inngest/components/icons/App';
import { IconSpinner } from '@inngest/components/icons/Spinner';

import AddAppButton from '@/components/App/AddAppButton';
import AppCard from '@/components/App/AppCard';
import { useInfoQuery } from '@/store/devApi';
import { useGetAppsQuery } from '@/store/generated';

export default function AppList() {
  const { data } = useGetAppsQuery(undefined, { pollingInterval: 1500 });
  const apps = data?.apps || [];

  const syncedApps = apps.filter((app) => app.connected === true);
  const numberOfSyncedApps = syncedApps.length;

  const memoizedAppCards = useMemo(() => {
    return apps.map((app) => {
      return <AppCard key={app?.id} app={app} />;
    });
  }, [apps]);

  const { data: info } = useInfoQuery();

  return (
    <div className="flex h-full flex-col overflow-y-scroll">
      <Header
        breadcrumb={[{ text: 'Apps' }]}
        infoIcon={
          <Info
            text="This is a list of all apps. We auto-detect apps that you have defined in specific ports."
            action={
              <Link href="https://www.inngest.com/docs/local-development#connecting-apps-to-the-dev-server">
                Go to specific ports.
              </Link>
            }
          />
        }
        action={
          <div className="flex items-center gap-5">
            {info?.isDiscoveryEnabled ? (
              <p className="text-btnPrimary flex items-center gap-2 text-sm leading-tight">
                <IconSpinner className="fill-btnPrimary" />
                Auto-detecting Apps
              </p>
            ) : null}
            <AddAppButton />
          </div>
        }
      />

      <div className="px-10 py-6">
        <div className="mb-4 flex items-center gap-3">
          <IconApp />
          <p className="text-subtle">
            {numberOfSyncedApps} / {apps.length} Apps Synced
          </p>
        </div>
        <div className="grid min-h-max grid-cols-1 gap-6 md:grid-cols-2">{memoizedAppCards}</div>
      </div>
    </div>
  );
}
