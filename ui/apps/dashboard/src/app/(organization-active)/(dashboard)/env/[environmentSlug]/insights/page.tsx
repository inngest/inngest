'use client';

import { useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useQuery } from 'urql';

import { SaveTabProvider } from '@/components/Insights/InsightsSQLEditor/SaveTabContext';
import { useDocumentShortcuts } from '@/components/Insights/InsightsSQLEditor/actions/handleShortcuts';
import { SchemasProvider } from '@/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/SchemasContext';
import { useInsightsTabManager } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { TabManagerProvider } from '@/components/Insights/InsightsTabManager/TabManagerContext';
import { QueryHelperPanel } from '@/components/Insights/QueryHelperPanel/QueryHelperPanel';
import { StoredQueriesProvider } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { useDeepLinkHandler } from '@/components/Insights/useDeepLinkHandler';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';

export default function InsightsPage() {
  const [isQueryHelperPanelVisible, setIsQueryHelperPanelVisible] = useState(true);
  const router = useRouter();
  const searchParams = useSearchParams();

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
      <InsightsContent
        actions={actions}
        activeSavedQueryId={activeSavedQueryId}
        activeTab={activeTab}
        isQueryHelperPanelVisible={isQueryHelperPanelVisible}
        router={router}
        searchParams={searchParams}
        tabManager={tabManager}
      />
    </StoredQueriesProvider>
  );
}

interface InsightsContentProps {
  actions: ReturnType<typeof useInsightsTabManager>['actions'];
  activeSavedQueryId: string | undefined;
  activeTab: ReturnType<typeof useInsightsTabManager>['tabs'][number] | undefined;
  isQueryHelperPanelVisible: boolean;
  router: ReturnType<typeof useRouter>;
  searchParams: ReturnType<typeof useSearchParams>;
  tabManager: JSX.Element;
}

function InsightsContent({
  actions,
  activeSavedQueryId,
  activeTab,
  isQueryHelperPanelVisible,
  router,
  searchParams,
  tabManager,
}: InsightsContentProps) {
  useDeepLinkHandler({ actions, activeSavedQueryId, router, searchParams });

  return (
    <SaveTabProvider>
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
    </SaveTabProvider>
  );
}
