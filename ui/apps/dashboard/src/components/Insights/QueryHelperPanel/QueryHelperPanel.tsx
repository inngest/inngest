'use client';

import { useCallback } from 'react';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { ulid } from 'ulid';

import type { TabManagerActions } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { QueryHelperPanelSection } from './QueryHelperPanelSection';
import { useRecentQueries, useSavedQueries, useTemplates } from './mock';
import type { Query } from './types';

const DEFAULT_ACCORDION_VALUES = ['TEMPLATES', 'RECENT QUERIES', 'SAVED QUERIES'];

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
      <AccordionList className="border-0" defaultValue={DEFAULT_ACCORDION_VALUES} type="multiple">
        <QueryHelperPanelSection
          onQuerySelect={handleQuerySelect}
          queries={templates}
          title="TEMPLATES"
        />
        <QueryHelperPanelSection
          onQuerySelect={handleQuerySelect}
          queries={recentQueries}
          title="RECENT QUERIES"
        />
        <QueryHelperPanelSection
          onQuerySelect={handleQuerySelect}
          queries={savedQueries}
          title="SAVED QUERIES"
        />
      </AccordionList>
    </div>
  );
}
