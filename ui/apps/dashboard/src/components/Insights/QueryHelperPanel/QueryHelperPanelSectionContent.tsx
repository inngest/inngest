'use client';

import type { TabConfig } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import type { Query, QuerySnapshot } from '../types';
import { QueryHelperPanelSectionContentNoData } from './QueryHelperPanelSectionContentNoData';
import { QueryHelperPanelSectionItem } from './QueryHelperPanelSectionItem';

export interface QueryHelperPanelSectionContentProps {
  activeTabId: string;
  onQuerySelect: (query: Query | QuerySnapshot) => void;
  queries: {
    data: undefined | Array<Query | QuerySnapshot>;
    error: undefined | string;
    isLoading: boolean;
  };
  sectionType: 'history' | 'saved';
  tabs: TabConfig[];
}

export function QueryHelperPanelSectionContent({
  activeTabId,
  onQuerySelect,
  queries,
  sectionType,
  tabs,
}: QueryHelperPanelSectionContentProps) {
  const { data, error, isLoading } = queries;

  if (isLoading) {
    return <QueryHelperPanelStaticMessage>Loading...</QueryHelperPanelStaticMessage>;
  }

  if (error) {
    return <QueryHelperPanelStaticMessage>{error}</QueryHelperPanelStaticMessage>;
  }

  if (!data?.length) {
    return (
      <QueryHelperPanelSectionContentNoData
        primary={sectionType === 'history' ? 'No query history' : 'No saved queries'}
        secondary={
          sectionType === 'history'
            ? 'You will find the last 10 queries that ran successfully here.'
            : 'Click the save query button to easily access your queries later.'
        }
        sectionType={sectionType}
      />
    );
  }

  return (
    <div className="flex flex-col gap-1">
      {data.map((query) => (
        <QueryHelperPanelSectionItem
          activeTabId={activeTabId}
          key={query.id}
          onQuerySelect={onQuerySelect}
          query={query}
          sectionType={sectionType}
          tabs={tabs}
        />
      ))}
    </div>
  );
}

function QueryHelperPanelStaticMessage({ children }: React.PropsWithChildren) {
  return (
    <div className="text-subtle w-full cursor-default overflow-x-hidden truncate text-ellipsis whitespace-nowrap rounded px-2 py-1.5 text-left text-sm font-medium opacity-60">
      {children}
    </div>
  );
}
