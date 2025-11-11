'use client';

import { useMemo } from 'react';
import { RiAddCircleFill, RiBookReadLine } from '@remixicon/react';

import { useTabManagerActions } from '@/components/Insights/InsightsTabManager/TabManagerContext';
import type { InsightsQueryStatement } from '@/gql/graphql';
import { QueryHelperPanelCollapsibleSection } from './QueryHelperPanelCollapsibleSection';
import { useStoredQueries } from './StoredQueriesContext';

interface QueryHelperPanelProps {
  activeSavedQueryId?: string;
}

export function QueryHelperPanel({ activeSavedQueryId }: QueryHelperPanelProps) {
  const { tabManagerActions } = useTabManagerActions();
  const { deleteQuery, deleteQuerySnapshot, querySnapshots, queries } = useStoredQueries();

  const sharedQueries = useMemo(() => limitQueriesByType('shared', queries), [queries]);
  const savedQueries = useMemo(() => limitQueriesByType('saved', queries), [queries]);

  return (
    <div className="border-subtle flex h-full w-full flex-col border-r">
      <div className="px-4 pb-1 pt-4">
        <h2 className="text-md mb-4 font-medium">Insights</h2>
        <div>
          <button
            className="hover:bg-canvasSubtle text-subtle hover:text-basis my-1 flex h-8 w-full flex-row items-center rounded px-1.5 text-left transition-colors"
            onClick={tabManagerActions.createNewTab}
          >
            <RiAddCircleFill className="text-primary-intense h-4 w-4" />
            <span className="text-primary-intense ml-2.5 text-sm font-medium leading-tight">
              New query
            </span>
          </button>
          <button
            className="hover:bg-canvasSubtle text-subtle hover:text-basis my-1 flex h-8 w-full flex-row items-center rounded px-1.5 text-left transition-colors"
            onClick={tabManagerActions.openTemplatesTab}
          >
            <RiBookReadLine className="h-4 w-4" />
            <span className="ml-2.5 text-sm font-medium leading-tight">Browse templates</span>
          </button>
        </div>
      </div>
      <div className="no-scrollbar flex-1 overflow-y-auto [&::-webkit-scrollbar]:hidden">
        <QueryHelperPanelCollapsibleSection
          activeSavedQueryId={activeSavedQueryId}
          onQueryDelete={deleteQuery}
          onQuerySelect={tabManagerActions.createTabFromQuery}
          queries={sharedQueries}
          title="Shared queries"
          sectionType="shared"
        />
        <QueryHelperPanelCollapsibleSection
          activeSavedQueryId={activeSavedQueryId}
          onQueryDelete={deleteQuery}
          onQuerySelect={tabManagerActions.createTabFromQuery}
          queries={savedQueries}
          title="Saved queries"
          sectionType="saved"
        />
        <QueryHelperPanelCollapsibleSection
          activeSavedQueryId={activeSavedQueryId}
          onQueryDelete={deleteQuerySnapshot}
          onQuerySelect={tabManagerActions.createTabFromQuery}
          queries={querySnapshots}
          title="Query history"
          sectionType="history"
        />
      </div>
    </div>
  );
}

type QueriesResponse = {
  data: undefined | Array<InsightsQueryStatement>;
  error: undefined | string;
  isLoading: boolean;
};

function limitQueriesByType(type: 'shared' | 'saved', queries: QueriesResponse) {
  return {
    ...queries,
    data: queries.data?.filter((query) => (type === 'shared' ? query.shared : !query.shared)),
  };
}
