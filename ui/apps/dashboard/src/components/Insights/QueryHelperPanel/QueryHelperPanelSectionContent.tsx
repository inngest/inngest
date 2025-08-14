'use client';

import { QueryHelperPanelSectionContentNoData } from './QueryHelperPanelSectionContentNoData';
import { QueryHelperPanelSectionItem } from './QueryHelperPanelSectionItem';
import { QueryHelperPanelStaticMessage } from './QueryHelperPanelStaticMessage';
import type { Query } from './types';

interface QueryHelperPanelSectionContentProps {
  onQuerySelect: (query: Query) => void;
  queries: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
  sectionType: 'history' | 'saved';
}

export function QueryHelperPanelSectionContent({
  onQuerySelect,
  queries,
  sectionType,
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
          key={query.id}
          onQuerySelect={onQuerySelect}
          query={query}
          sectionType={sectionType}
        />
      ))}
    </div>
  );
}
