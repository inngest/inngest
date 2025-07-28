'use client';

import { Header } from '@inngest/components/Header/Header';

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

function InsightsSQLEditor() {
  return null;
}

function InsightsDataTable() {
  return null;
}
