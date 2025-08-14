'use client';

import { useState } from 'react';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useInsightsTabManager } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { TabManagerProvider } from '@/components/Insights/InsightsTabManager/TabManagerContext';
import { QueryHelperPanel } from '@/components/Insights/QueryHelperPanel';

// NOTE: The usage of isQueryHelperPanelVisible causes re-fetching when toggled,
// but this is okay for now because the fetching shouldn't be that low anyway.

function InsightsContent() {
  const [isQueryHelperPanelVisible, setIsQueryHelperPanelVisible] = useState(true);

  const { actions, tabManager } = useInsightsTabManager({
    isQueryHelperPanelVisible,
    onToggleQueryHelperPanelVisibility: () => setIsQueryHelperPanelVisible((visible) => !visible),
  });

  return (
    <TabManagerProvider actions={actions}>
      <div className="flex h-full w-full flex-1 overflow-hidden">
        {isQueryHelperPanelVisible && (
          <div className="w-[240px] flex-shrink-0">
            <QueryHelperPanel />
          </div>
        )}
        <div className="flex h-full w-full flex-1 flex-col overflow-hidden">{tabManager}</div>
      </div>
    </TabManagerProvider>
  );
}

export default function InsightsPage() {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');
  if (!isInsightsEnabled) return null;

  return <InsightsContent />;
}
