'use client';

import { useInsightsQueryContext } from '../context';
import { ResultsState } from './ResultsState';
import { EmptyState } from './states/EmptyState';
import { ErrorState } from './states/ErrorState';
import { LoadingState } from './states/LoadingState';

export function InsightsDataTable() {
  const { state } = useInsightsQueryContext();

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {(() => {
        switch (state) {
          case 'error':
            return <ErrorState />;
          case 'initial':
            return <EmptyState />;
          case 'loading':
            return <LoadingState />;
          case 'fetchingMore':
          case 'fetchMoreError':
          case 'success': {
            return <ResultsState />;
          }
        }
      })()}
    </div>
  );
}
