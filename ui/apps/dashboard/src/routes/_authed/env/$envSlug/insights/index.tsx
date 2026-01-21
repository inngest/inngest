import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { useQuery } from 'urql';

import { createFileRoute } from '@tanstack/react-router';
import { useRef, useState } from 'react';
import {
  useInsightsTabManager,
  type TabManagerActions,
} from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { useDocumentShortcuts } from '@/components/Insights/InsightsSQLEditor/actions/handleShortcuts';
import {
  StoredQueriesProvider,
  useStoredQueries,
} from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
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

// Initial placeholder actions used before real actions are available from useInsightsTabManager
// Defined outside component to avoid recreation on every render
const INITIAL_TAB_ACTIONS: TabManagerActions = {
  breakQueryAssociation: () => {},
  closeTab: () => {},
  createNewTab: () => {},
  createTabFromQuery: () => {},
  focusTab: () => {},
  openTemplatesTab: () => {},
  updateTab: () => {},
};

function InsightsComponent() {
  const [isQueryHelperPanelVisible, setIsQueryHelperPanelVisible] =
    useState(true);

  const [{ data: entitlementsData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });
  const historyWindow = entitlementsData?.account.entitlements.history.limit;

  const search = Route.useSearch();
  const deepLinkQueryId =
    typeof search.query_id === 'string' && search.query_id.length > 0
      ? search.query_id
      : undefined;

  // Create a ref for actions that will be populated inside InsightsWithTabManager
  // This allows StoredQueriesProvider to use the latest actions without recreating the provider
  const actionsRef = useRef<TabManagerActions>(INITIAL_TAB_ACTIONS);

  return (
    <StoredQueriesProvider tabManagerActionsRef={actionsRef}>
      <InsightsWithTabManager
        historyWindow={historyWindow}
        isQueryHelperPanelVisible={isQueryHelperPanelVisible}
        onToggleQueryHelperPanelVisibility={() =>
          setIsQueryHelperPanelVisible((visible) => !visible)
        }
        deepLinkQueryId={deepLinkQueryId}
        actionsRef={actionsRef}
      />
    </StoredQueriesProvider>
  );
}

interface InsightsWithTabManagerProps {
  historyWindow?: number;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  deepLinkQueryId?: string;
  actionsRef: React.MutableRefObject<TabManagerActions>;
}

function InsightsWithTabManager({
  historyWindow,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  deepLinkQueryId,
  actionsRef,
}: InsightsWithTabManagerProps) {
  const { isSavedQueriesFetching } = useStoredQueries();

  const { actions, activeTabId, tabManager, tabs } = useInsightsTabManager({
    historyWindow,
    isQueryHelperPanelVisible,
    onToggleQueryHelperPanelVisibility,
    isSavedQueriesFetching,
    deepLinkQueryId,
  });

  // Update the ref with real actions so StoredQueriesProvider can use them
  actionsRef.current = actions;

  useDocumentShortcuts([
    {
      combo: { alt: true, code: 'KeyT', metaOrCtrl: true },
      handler: actions.createNewTab,
    },
  ]);

  const activeTab = tabs.find((t) => t.id === activeTabId);
  const activeSavedQueryId = activeTab?.savedQueryId;

  return (
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
