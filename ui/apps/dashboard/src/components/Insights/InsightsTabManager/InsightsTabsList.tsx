'use client';

import { ulid } from 'ulid';

import { InsightsTab } from './InsightsTab';
import { InsightsTabActionIcon } from './InsightsTabActionIcon';
import type { TabConfig, TabManagerActions } from './InsightsTabManager';

interface InsightsTabsListProps {
  actions: TabManagerActions;
  activeTabId: string;
  hide?: boolean;
  tabs: TabConfig[];
}

export function InsightsTabsList({ actions, activeTabId, hide, tabs }: InsightsTabsListProps) {
  if (hide) return null;

  return (
    <div className="border-subtle overflow-x-auto border-b [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      <div className="flex items-center">
        <div className="flex flex-shrink-0 items-center">
          <InsightsTabActionIcon
            isActive={activeTabId === '__home'}
            isFirst={true}
            onClick={() => actions.focusTab('__home')}
            tooltip="Home"
            type="home"
          />
          {tabs
            .filter((tab) => tab.id !== '__home')
            .map((tab) => (
              <div key={tab.id} className="-ml-px">
                <InsightsTab
                  isActive={tab.id === activeTabId}
                  isFirst={false}
                  name={tab.name}
                  onClick={() => {
                    actions.focusTab(tab.id);
                  }}
                  onClose={() => actions.closeTab(tab.id)}
                  showCloseButton={true}
                />
              </div>
            ))}
          <div className="-ml-px">
            <InsightsTabActionIcon
              onClick={() => {
                actions.createTab({
                  id: ulid(),
                  name: 'Untitled query',
                  query: '',
                  type: 'new',
                });
              }}
              type="add"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
