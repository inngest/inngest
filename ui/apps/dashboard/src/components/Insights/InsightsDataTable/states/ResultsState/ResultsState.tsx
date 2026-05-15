import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { NoResults } from './NoResults';
import { ResultsTable } from './ResultsTable';
import { DiagnosticsBanner } from '../../DiagnosticsBanner';

export function ResultsState() {
  const { data } = useInsightsStateMachineContext();

  return (
    <>
      {data?.diagnostics?.length ? <DiagnosticsBanner /> : null}
      {data?.rows?.length ? <ResultsTable /> : <NoResults />}
    </>
  );
}
