'use client';

import { Header } from '@inngest/components/Header/Header';

import { DataTable } from '@/components/Insights/DataTable';
import { SQLEditor } from '@/components/Insights/SQLEditor';
import { useInsightsQuery } from '@/components/Insights/useInsightsQuery';

type InsightsPageProps = {
  params: {
    environmentSlug: string;
  };
};

export default function InsightsPage({ params: { environmentSlug } }: InsightsPageProps) {
  const {
    executeQuery,
    isLoading,
    result: { data },
  } = useInsightsQuery();

  return (
    <>
      <Header breadcrumb={[{ text: 'Insights' }]} />
      <main className="bg-canvasBase no-scrollbar text-basis flex-1 overflow-hidden focus-visible:outline-none">
        <div className="flex h-full">
          <div className="flex w-full flex-col">
            <SQLEditor isLoading={isLoading} onRunQuery={executeQuery} />
            <DataTable data={data} isLoading={isLoading} />
          </div>
        </div>
      </main>
    </>
  );
}
