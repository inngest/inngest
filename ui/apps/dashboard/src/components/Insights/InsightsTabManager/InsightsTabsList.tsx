'use client';

import { RiAddLine } from '@remixicon/react';
import { ulid } from 'ulid';

import { InsightsTab } from './InsightsTab';
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
    <div className="flex items-center overflow-x-auto [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      <div className="flex flex-shrink-0 items-center">
        {tabs.map((tab, index) => (
          <div key={tab.id} className={index > 0 ? '-ml-px' : ''}>
            <InsightsTab
              isActive={tab.id === activeTabId}
              isFirst={index === 0}
              name={tab.name}
              onClick={() => {
                actions.focusTab(tab.id);
              }}
              onClose={tab.id !== '__home' ? () => actions.closeTab(tab.id) : undefined}
              showCloseButton={tab.id !== '__home'}
            />
          </div>
        ))}
        <div className="-ml-px">
          <button
            className="bg-canvasBase border-subtle hover:bg-canvasMuted hover:border-muted border-b-subtle box-border flex h-12 w-12 items-center justify-center border-b-2 border-l border-r transition-colors"
            onClick={() => {
              actions.createTab({
                id: ulid(),
                name: 'Untitled query',
                query: '',
                type: 'new',
              });
            }}
          >
            <RiAddLine className="h-4 w-4 text-slate-500" />
          </button>
        </div>
      </div>
    </div>
  );
}
