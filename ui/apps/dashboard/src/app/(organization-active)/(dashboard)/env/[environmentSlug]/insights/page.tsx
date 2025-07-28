'use client';

import { Header } from '@inngest/components/Header/Header';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { InsightsDataTable } from '@/components/Insights/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditor';

export default function InsightsPage() {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');

  if (!isInsightsEnabled) return null;

  return (
    <>
      <Header breadcrumb={[{ text: 'Insights' }]} />
      <main className="grid h-full w-full flex-1 grid-rows-[3fr_5fr] gap-0">
        <div className="min-h-[240px]">
          <InsightsSQLEditor />
        </div>
        <InsightsDataTable />
      </main>
    </>
  );
}
