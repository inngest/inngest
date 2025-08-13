'use client';

import { Tabs } from '@inngest/components/Tabs';
import { RiAddLine, RiCodeLine, RiContractLeftLine, RiExpandRightLine } from '@remixicon/react';
import { ulid } from 'ulid';

import type { TabConfig, TabManagerActions } from './InsightsTabManager';

interface InsightsTabsListProps {
  actions: TabManagerActions;
  activeTabId: string;
  hide?: boolean;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: TabConfig[];
}

export function InsightsTabsList({
  actions,
  activeTabId,
  hide,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  tabs,
}: InsightsTabsListProps) {
  if (hide) return null;

  return (
    <Tabs
      value={activeTabId}
      onValueChange={actions.focusTab}
      onClose={actions.closeTab}
      defaultIconBefore={<RiCodeLine size={16} />}
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
          <Tabs.Tab key={tab.id} value={tab.id}>
            {tab.name}
          </Tabs.Tab>
        ))}
        <Tabs.IconTab
          icon={<RiAddLine size={16} />}
          onClick={() => {
            actions.createTab({
              id: ulid(),
              name: 'Untitled query',
              query: '',
              type: 'new',
            });
          }}
          title="Add new tab"
        />
      </Tabs.List>
    </Tabs>
  );
}
