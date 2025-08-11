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
      tabManagerActions.createTab(ulid(), query.name);
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
