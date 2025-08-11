'use client';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useInsightsTabManager } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { QueryHelperPanel } from '@/components/Insights/QueryHelperPanel';

function InsightsContent() {
  const { actions, tabManager } = useInsightsTabManager();

  return (
    <div className="flex h-full w-full flex-1 overflow-hidden">
      <div className="w-[280px] flex-shrink-0">
        <QueryHelperPanel tabManagerActions={actions} />
      </div>
      <div className="flex h-full w-full flex-1 flex-col overflow-hidden">{tabManager}</div>
    </div>
  );
}

export default function InsightsPage() {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');
  if (!isInsightsEnabled) return null;

  return <InsightsContent />;
}
