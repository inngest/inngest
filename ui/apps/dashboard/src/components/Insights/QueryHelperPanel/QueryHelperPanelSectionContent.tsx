'use client';

import type { InsightsQueryStatement } from '@/gql/graphql';
import type { QuerySnapshot } from '../types';
import { QueryHelperPanelSectionContentNoData } from './QueryHelperPanelSectionContentNoData';
import { QueryHelperPanelSectionItem } from './QueryHelperPanelSectionItem';

export interface QueryHelperPanelSectionContentProps {
  activeSavedQueryId?: string;
  onQueryDelete: (queryId: string) => void;
  onQuerySelect: (query: InsightsQueryStatement | QuerySnapshot) => void;
  queries: {
    data: undefined | Array<InsightsQueryStatement | QuerySnapshot>;
    error: undefined | string;
    isLoading: boolean;
  };
  sectionType: 'history' | 'saved';
}

export function QueryHelperPanelSectionContent({
  activeSavedQueryId,
  onQueryDelete,
  onQuerySelect,
  queries,
  sectionType,
}: QueryHelperPanelSectionContentProps) {
  const { data, error, isLoading } = queries;

  if (isLoading && !data?.length) {
    return <QueryHelperPanelStaticMessage>Loading...</QueryHelperPanelStaticMessage>;
  }

  if (error && !data?.length) {
    return <QueryHelperPanelStaticMessage>Failed to load queries</QueryHelperPanelStaticMessage>;
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
          activeSavedQueryId={activeSavedQueryId}
          key={query.id}
          onQueryDelete={onQueryDelete}
          onQuerySelect={onQuerySelect}
          query={query}
          sectionType={sectionType}
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
