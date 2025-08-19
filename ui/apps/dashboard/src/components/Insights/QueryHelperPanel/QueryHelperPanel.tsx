'use client';

import { useMemo } from 'react';
import { RiAddCircleFill, RiBookReadLine } from '@remixicon/react';

import { useTabManagerActions } from '@/components/Insights/InsightsTabManager/TabManagerContext';
import { getOrderedQuerySnapshots, getOrderedSavedQueries } from '../queries';
import { QueryHelperPanelCollapsibleSection } from './QueryHelperPanelCollapsibleSection';
import { useStoredQueries } from './StoredQueriesContext';

interface QueryHelperPanelProps {
  activeTabId: string;
}

export function QueryHelperPanel({ activeTabId }: QueryHelperPanelProps) {
  const { tabManagerActions } = useTabManagerActions();
  const { deleteQuery, deleteQuerySnapshot, queries, querySnapshots } = useStoredQueries();

  const savedQueries = useMemo(() => {
    return temporarilyWrapData(getOrderedSavedQueries(queries));
  }, [queries]);

  const orderedQuerySnapshots = useMemo(() => {
    return temporarilyWrapData(getOrderedQuerySnapshots(querySnapshots));
  }, [querySnapshots]);

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
              New insight
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
          activeTabId={activeTabId}
          onQueryDelete={deleteQuery}
          onQuerySelect={tabManagerActions.createTabFromQuery}
          queries={savedQueries}
          title="Saved queries"
          sectionType="saved"
        />
        <QueryHelperPanelCollapsibleSection
          activeTabId={activeTabId}
          onQueryDelete={deleteQuerySnapshot}
          onQuerySelect={tabManagerActions.createTabFromQuery}
          queries={orderedQuerySnapshots}
          title="Query history"
          sectionType="history"
        />
      </div>
    </div>
  );
}

// TODO: Use real error, loading values when data is fetched from the server.
function temporarilyWrapData<T>(data: T): {
  data: T;
  error: undefined;
  isLoading: boolean;
} {
  return { data, error: undefined, isLoading: false };
}
