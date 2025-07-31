'use client';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { NoResults } from './NoResults';
import { ResultsTable } from './ResultsTable';

export function ResultsState() {
  const { data } = useInsightsStateMachineContext();

  if (!data?.entries.length) return <NoResults />;

  return <ResultsTable />;
}
