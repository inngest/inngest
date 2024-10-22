'use server';

import { Header } from '@inngest/components/Header/Header';
import { RefreshButton } from '@inngest/components/Refresh/RefreshButton';

import { MetricsActionMenu } from '@/components/Metrics/ActionMenu';
import { Dashboard } from '@/components/Metrics/Dashboard';

type MetricsProps = {
  params: {
    environmentSlug: string;
  };
};

export default async function MetricsPage({ params: { environmentSlug: envSlug } }: MetricsProps) {
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
      <div id="chart-tooltip" className="z-[1000]" />
      <div className="bg-canvasSubtle mx-auto flex h-full w-full flex-col">
        <Dashboard envSlug={envSlug} />
      </div>
    </>
  );
}
