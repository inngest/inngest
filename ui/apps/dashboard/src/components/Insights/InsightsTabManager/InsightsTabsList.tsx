'use client';

import Tabs from '@inngest/components/Tabs/Tabs';
import {
  RiAddLine,
  RiBookReadLine,
  RiCodeLine,
  RiContractLeftLine,
  RiExpandRightLine,
} from '@remixicon/react';

import type { TabConfig } from './InsightsTabManager';
import { useTabManagerActions } from './TabManagerContext';
import { TEMPLATES_TAB } from './constants';

interface InsightsTabsListProps {
  activeTabId: string;
  hide?: boolean;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: TabConfig[];
}

export function InsightsTabsList({
  activeTabId,
  hide,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  tabs,
}: InsightsTabsListProps) {
  const { tabManagerActions } = useTabManagerActions();

  if (hide) return null;

  return (
    <Tabs
      defaultIconBefore={<RiCodeLine size={16} />}
      onClose={tabManagerActions.closeTab}
      onValueChange={tabManagerActions.focusTab}
      value={activeTabId}
    >
      <Tabs.List>
        <Tabs.IconTab
          icon={
            isQueryHelperPanelVisible ? (
              <RiContractLeftLine size={16} />
            ) : (
              <RiExpandRightLine size={16} />
            )
          }
          onClick={onToggleQueryHelperPanelVisibility}
          title={`${isQueryHelperPanelVisible ? 'Hide' : 'Show'} sidebar`}
        />
        {tabs.map((tab) => (
          <Tabs.Tab
            iconBefore={tab.id === TEMPLATES_TAB.id ? <RiBookReadLine size={16} /> : undefined}
            key={tab.id}
            value={tab.id}
          >
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
