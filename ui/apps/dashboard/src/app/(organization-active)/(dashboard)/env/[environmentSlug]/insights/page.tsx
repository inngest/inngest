'use client';

import { useState } from 'react';
import { useQuery } from 'urql';

import { useDocumentShortcuts } from '@/components/Insights/InsightsSQLEditor/actions/handleShortcuts';
import { SchemasProvider } from '@/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/SchemasContext';
import { useInsightsTabManager } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { TabManagerProvider } from '@/components/Insights/InsightsTabManager/TabManagerContext';
import { QueryHelperPanel } from '@/components/Insights/QueryHelperPanel/QueryHelperPanel';
import { StoredQueriesProvider } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';

export default function InsightsPage() {
  const [isQueryHelperPanelVisible, setIsQueryHelperPanelVisible] = useState(true);

  const [{ data: entitlementsData }] = useQuery({ query: GetAccountEntitlementsDocument });
  const historyWindow = entitlementsData?.account.entitlements.history.limit;

  const { actions, activeTabId, tabManager, tabs } = useInsightsTabManager({
    historyWindow,
    isQueryHelperPanelVisible,
    onToggleQueryHelperPanelVisibility: () => setIsQueryHelperPanelVisible((visible) => !visible),
  });

  useDocumentShortcuts([
    {
      combo: { alt: true, code: 'KeyT', metaOrCtrl: true },
      handler: actions.createNewTab,
    },
  ]);

  const activeTab = tabs.find((t) => t.id === activeTabId);
  const activeSavedQueryId = activeTab?.savedQueryId;

  return (
    <StoredQueriesProvider tabManagerActions={actions}>
      <TabManagerProvider actions={actions} activeTab={activeTab}>
        <SchemasProvider>
          <div className="flex h-full w-full flex-1 overflow-hidden">
            {isQueryHelperPanelVisible && (
              <div className="w-[240px] flex-shrink-0">
                <QueryHelperPanel activeSavedQueryId={activeSavedQueryId} />
              </div>
            )}
            <div className="flex h-full w-full flex-1 flex-col overflow-hidden">{tabManager}</div>
          </div>
        </SchemasProvider>
      </TabManagerProvider>
    </StoredQueriesProvider>
  );
}
