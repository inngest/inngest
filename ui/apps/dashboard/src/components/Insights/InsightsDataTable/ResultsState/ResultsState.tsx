'use client';

import { useInsightsQueryContext } from '../../context';
import { NoResults } from './NoResults';
import { ResultsTable } from './ResultsTable';

export function ResultsState() {
  const { data } = useInsightsQueryContext();

  if (!data?.entries.length) return <NoResults />;

  return <ResultsTable />;
}
