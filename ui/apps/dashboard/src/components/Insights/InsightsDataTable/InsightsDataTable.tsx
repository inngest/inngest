'use client';

import { useInsightsQueryContext } from '../context';
import { ResultsState } from './ResultsState';
import { EmptyState } from './states/EmptyState';
import { ErrorState } from './states/ErrorState';
import { LoadingState } from './states/LoadingState';
import { NoResultsState } from './states/NoResultsState';

export function InsightsDataTable() {
  const { data, state } = useInsightsQueryContext();

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {(() => {
        switch (state) {
          case 'error':
            return <ErrorState />;
          case 'fetchingMore':
            return <ResultsState />;
          case 'initial':
            return <EmptyState />;
          case 'loading':
            return <LoadingState />;
          case 'fetchMoreError':
          case 'success': {
            if (!data?.entries.length) return <NoResultsState />;
            return <ResultsState />;
          }
        }
      })()}
    </div>
  );
}
