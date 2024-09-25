'use server';

import { Header } from '@inngest/components/Header/Header';
import { RefreshButton } from '@inngest/components/Refresh/RefreshButton';

import { MetricsActionMenu } from '@/components/Metrics/ActionMenu';
import { Dashboard } from '@/components/Metrics/Dashboard';
import { getMetricsLookups, preloadMetricsLookups } from '@/components/Metrics/data';

type MetricsProps = {
  params: {
    environmentSlug: string;
  };
};

export default async function MetricsPage({ params: { environmentSlug: envSlug } }: MetricsProps) {
  preloadMetricsLookups(envSlug);
  const {
    envBySlug: {
      apps,
      workflows: { data: functions },
    },
  } = await getMetricsLookups(envSlug);

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Metrics' }]}
        action={
          <div className="flex flex-row items-center justify-end gap-x-1">
            <RefreshButton />
            <MetricsActionMenu />
          </div>
        }
      />
      <div className="bg-canvasSubtle mx-auto flex h-full w-full flex-col">
        <Dashboard
          apps={apps
            .filter(({ isArchived }) => isArchived === false)
            .map((app: { id: string; externalID: string }) => ({
              id: app.id,
              name: app.externalID,
            }))}
          functions={functions}
        />
      </div>
    </>
  );
}
