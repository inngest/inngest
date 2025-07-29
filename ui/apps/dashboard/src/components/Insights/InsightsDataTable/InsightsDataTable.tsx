'use client';

import { useInsightsQueryContext } from '../context';
import { EmptyState } from './EmptyState';
import { ErrorState } from './ErrorState';
import { LoadingState } from './LoadingState';
import { NoResultsState } from './NoResultsState';
import { ResultsTable } from './ResultsTable';

export function InsightsDataTable() {
  const { data, state } = useInsightsQueryContext();

  switch (state) {
    case 'loading':
      return <LoadingState />;
    case 'success':
      return data?.rows.length ? <ResultsTable /> : <NoResultsState />;
    case 'error':
      return <ErrorState />;
    default:
      return <EmptyState />;
  }
}
