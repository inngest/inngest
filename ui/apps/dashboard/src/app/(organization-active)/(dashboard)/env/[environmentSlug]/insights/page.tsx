'use client';

import { Header } from '@inngest/components/Header/Header';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useInsightsTabManager } from '@/components/Insights/InsightsTabManager/InsightsTabManager';

function InsightsContent() {
  const { tabManager } = useInsightsTabManager();

  return (
    <>
      <Header breadcrumb={[{ text: 'Insights' }]} />
      {/* TODO: Add templates, recent queries, saved queries sidepanel */}
      {tabManager}
    </>
  );
}

export default function InsightsPage() {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');
  if (!isInsightsEnabled) return null;

  return <InsightsContent />;
}
