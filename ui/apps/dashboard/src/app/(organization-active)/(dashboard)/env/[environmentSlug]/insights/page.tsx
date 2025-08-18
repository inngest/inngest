'use client';

import { useState } from 'react';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useInsightsTabManager } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { TabManagerProvider } from '@/components/Insights/InsightsTabManager/TabManagerContext';
import { QueryHelperPanel } from '@/components/Insights/QueryHelperPanel/QueryHelperPanel';
import { StoredQueriesProvider } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';

function InsightsContent() {
  const [isQueryHelperPanelVisible, setIsQueryHelperPanelVisible] = useState(true);

  const { actions, tabManager } = useInsightsTabManager({
    isQueryHelperPanelVisible,
    onToggleQueryHelperPanelVisibility: () => setIsQueryHelperPanelVisible((visible) => !visible),
  });

  return (
    <StoredQueriesProvider tabManagerActions={actions}>
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
    </StoredQueriesProvider>
  );
}

export default function InsightsPage() {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');
  if (!isInsightsEnabled) return null;

  return <InsightsContent />;
}
