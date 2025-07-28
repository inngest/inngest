'use client';

import { Header } from '@inngest/components/Header/Header';

import { InsightsDataTable } from '@/components/Insights/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor';

export default function InsightsPage() {
  return (
    <>
      <Header breadcrumb={[{ text: 'Insights' }]} />
      <main className="flex h-full w-full flex-1 flex-col">
        <InsightsSQLEditor />
        <InsightsDataTable />
      </main>
    </>
  );
}
