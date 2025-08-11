'use client';

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
  sectionType?: 'history' | 'saved';
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
    return <QueryHelperPanelStaticMessage>No queries found</QueryHelperPanelStaticMessage>;
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
