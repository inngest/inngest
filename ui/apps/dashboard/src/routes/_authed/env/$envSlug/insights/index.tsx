import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { useQuery } from 'urql';

import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import {
  useInsightsTabManager,
  type TabManagerActions,
} from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { useDocumentShortcuts } from '@/components/Insights/InsightsSQLEditor/actions/handleShortcuts';
import { StoredQueriesProvider } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { SaveTabProvider } from '@/components/Insights/InsightsSQLEditor/SaveTabContext';
import { TabManagerProvider } from '@/components/Insights/InsightsTabManager/TabManagerContext';
import { SchemasProvider } from '@/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/SchemasContext';
import { QueryHelperPanel } from '@/components/Insights/QueryHelperPanel/QueryHelperPanel';
import { useDeepLinkHandler } from '@/components/Insights/useDeepLinkHandler';

export type InsightsSearchParams = {
  query_id?: string;
};

export const Route = createFileRoute('/_authed/env/$envSlug/insights/')({
  component: InsightsComponent,
  validateSearch: (search: Record<string, unknown>): InsightsSearchParams => {
    return {
      query_id: search?.query_id as string | undefined,
    };
  },
});

function InsightsComponent() {
  const [isQueryHelperPanelVisible, setIsQueryHelperPanelVisible] =
    useState(true);

  const [{ data: entitlementsData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });
  const historyWindow = entitlementsData?.account.entitlements.history.limit;

  const { actions, activeTabId, tabManager, tabs } = useInsightsTabManager({
    historyWindow,
    isQueryHelperPanelVisible,
    onToggleQueryHelperPanelVisibility: () =>
      setIsQueryHelperPanelVisible((visible) => !visible),
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
      <SaveTabProvider>
        <TabManagerProvider actions={actions} activeTab={activeTab}>
          <SchemasProvider>
            <InsightsContentWithDeepLink
              isQueryHelperPanelVisible={isQueryHelperPanelVisible}
              activeSavedQueryId={activeSavedQueryId}
              tabManager={tabManager}
              actions={actions}
            />
          </SchemasProvider>
        </TabManagerProvider>
      </SaveTabProvider>
    </StoredQueriesProvider>
  );
}

function InsightsContentWithDeepLink({
  isQueryHelperPanelVisible,
  activeSavedQueryId,
  tabManager,
  actions,
}: {
  isQueryHelperPanelVisible: boolean;
  activeSavedQueryId: string | undefined;
  tabManager: JSX.Element;
  actions: TabManagerActions;
}) {
  const navigate = Route.useNavigate();
  const search = Route.useSearch();

  // Handle deep linking with query_id parameter
  useDeepLinkHandler({
    actions,
    activeSavedQueryId,
    navigate,
    search,
  });

  return (
    <div className="flex h-full w-full flex-1 overflow-hidden">
      {isQueryHelperPanelVisible && (
        <div className="w-[240px] flex-shrink-0">
          <QueryHelperPanel activeSavedQueryId={activeSavedQueryId} />
        </div>
      )}
      <div className="flex h-full w-full flex-1 flex-col overflow-hidden">
        {tabManager}
      </div>
    </div>
  );
}
