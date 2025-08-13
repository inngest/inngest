'use client';

import { useCallback } from 'react';
import { RiAddCircleFill, RiBookReadLine } from '@remixicon/react';
import { ulid } from 'ulid';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { QueryHelperPanelCollapsibleSection } from './QueryHelperPanelCollapsibleSection';
import { useRecentQueries, useSavedQueries, useTemplates } from './mock';
import type { Query } from './types';

type QueryHelperPanelProps = {
  tabManagerActions: TabManagerActions;
};

export function QueryHelperPanel({ tabManagerActions }: QueryHelperPanelProps) {
  const recentQueries = useRecentQueries();
  const savedQueries = useSavedQueries();
  const templates = useTemplates();

  // TODO: Implement intended logic.
  const handleQuerySelect = useCallback(
    (query: Query) => {
      switch (query.type) {
        case 'recent':
        case 'template': {
          // Use a new ID to effectively clone the query.
          tabManagerActions.createTab({ ...query, id: ulid(), name: 'Untitled query' });
          break;
        }
        case 'saved': {
          // Allow only one instance of a saved query to be open at a time.
          const existingTabId = tabManagerActions.getTabIdForSavedQuery(query.id);
          if (existingTabId) {
            tabManagerActions.focusTab(existingTabId);
            return;
          }

          tabManagerActions.createTab(query);
          break;
        }
        default: {
          console.warn('Attempted to create a tab for an unknown query type', query);
          break;
        }
      }
    },
    [tabManagerActions]
  );

  return (
    <div className="border-subtle flex h-full w-full flex-col border-r">
      <div className="px-4 pb-1 pt-4">
        <h2 className="text-md mb-4 font-medium">Insights</h2>
        <div>
          <button
            className="hover:bg-canvasSubtle text-subtle hover:text-basis my-1 flex h-8 w-full flex-row items-center rounded px-1.5 text-left transition-colors"
            onClick={() => {
              tabManagerActions.createTab({
                id: ulid(),
                name: 'Untitled query',
                query: '',
                type: 'new',
              });
            }}
          >
            <RiAddCircleFill className="text-primary-intense h-4 w-4" />
            <span className="text-primary-intense ml-2.5 text-sm font-medium leading-tight">
              New insight
            </span>
          </button>
          <button
            className="hover:bg-canvasSubtle text-subtle hover:text-basis my-1 flex h-8 w-full flex-row items-center rounded px-1.5 text-left transition-colors"
            onClick={tabManagerActions.focusOrCreateTemplatesTab}
          >
            <RiBookReadLine className="h-4 w-4" />
            <span className="ml-2.5 text-sm font-medium leading-tight">Browse templates</span>
          </button>
        </div>
      </div>
      <div className="flex-1">
        <QueryHelperPanelCollapsibleSection
          onQuerySelect={handleQuerySelect}
          queries={savedQueries}
          title="Saved queries"
          sectionType="saved"
        />
        <QueryHelperPanelCollapsibleSection
          onQuerySelect={handleQuerySelect}
          queries={recentQueries}
          title="Query history"
          sectionType="history"
        />
      </div>
    </div>
  );
}
