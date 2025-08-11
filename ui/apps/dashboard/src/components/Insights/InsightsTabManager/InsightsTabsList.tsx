'use client';

import TabCards from '@inngest/components/TabCards/TabCards';
import { ulid } from 'ulid';

import type { TabConfig, TabManagerActions } from './InsightsTabManager';

// TODO: Complete implementation and remove "hide" prop.

interface InsightsTabsListProps {
  actions: TabManagerActions;
  activeTabId: string;
  hide?: boolean;
  tabs: TabConfig[];
}

export function InsightsTabsList({ actions, activeTabId, hide, tabs }: InsightsTabsListProps) {
  if (hide) return null;

  const handleTabChange = (value: string) => {
    // Don't change the active tab, just create a new one
    if (value === '__plus') return;

    actions.focusTab(value);
  };

  const handlePlusClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    actions.createTab({
      id: ulid(),
      name: 'Untitled query',
      query: '',
      type: 'new',
    });
  };

  return (
    <TabCards value={activeTabId} onValueChange={handleTabChange}>
      <TabCards.ButtonList>
        {tabs.map((tab) => (
          <TabCards.Button key={tab.id} value={tab.id}>
            {tab.name}
          </TabCards.Button>
        ))}
        <TabCards.Button key="__plus" value="__plus" onClick={handlePlusClick}>
          +
        </TabCards.Button>
      </TabCards.ButtonList>
    </TabCards>
  );
}
