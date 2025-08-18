'use client';

import Tabs from '@inngest/components/Tabs/Tabs';
import {
  RiAddLine,
  RiBookReadLine,
  RiCodeLine,
  RiContractLeftLine,
  RiExpandRightLine,
} from '@remixicon/react';

import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import type { TabConfig } from './InsightsTabManager';
import { useTabManagerActions } from './TabManagerContext';
import { TEMPLATES_TAB } from './constants';

interface InsightsTabsListProps {
  activeTabId: string;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: TabConfig[];
}

export function InsightsTabsList({
  activeTabId,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  tabs,
}: InsightsTabsListProps) {
  const { tabManagerActions } = useTabManagerActions();

  const ActionTabIcon = isQueryHelperPanelVisible ? RiContractLeftLine : RiExpandRightLine;

  return (
    <Tabs
      defaultIconBefore={<RiCodeLine size={16} />}
      onClose={tabManagerActions.closeTab}
      onValueChange={tabManagerActions.focusTab}
      value={activeTabId}
    >
      <Tabs.List>
        <Tabs.IconTab
          icon={<ActionTabIcon size={16} />}
          onClick={onToggleQueryHelperPanelVisibility}
          title={`${isQueryHelperPanelVisible ? 'Hide' : 'Show'} sidebar`}
        />
        {tabs.map((tab) => (
          <Tabs.Tab iconBefore={<IndicatorTabIcon tab={tab} />} key={tab.id} value={tab.id}>
            {tab.name}
          </Tabs.Tab>
        ))}
        <Tabs.IconTab
          icon={<RiAddLine size={16} />}
          onClick={tabManagerActions.createNewTab}
          title="Add new tab"
        />
      </Tabs.List>
    </Tabs>
  );
}

function IndicatorTabIcon({ tab }: { tab: TabConfig }) {
  const { queries } = useStoredQueries();

  if (tab.id === TEMPLATES_TAB.id) return <RiBookReadLine size={16} />;

  const savedQuery = tab.savedQueryId ? queries[tab.savedQueryId] : undefined;
  if (savedQuery === undefined) return <RiCodeLine size={16} />;

  const hasChanged = savedQuery.name !== tab.name || savedQuery.query !== tab.query;
  if (!hasChanged) return <RiCodeLine size={16} />;

  return <div className="h-2 w-2 rounded-full bg-amber-500" />;
}
