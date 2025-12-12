import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';

export function ResultsTableFooter() {
  const { data, status } = useInsightsStateMachineContext();

  if (status !== 'success') return null;
  if (!assertData(data)) return null;

  return (
    <div className="border-subtle flex h-[45px] items-center justify-between border-t py-0">
      <div className="text-muted pl-3 text-sm">
        {`${data.rows.length} ${data.rows.length === 1 ? 'row' : 'rows'}`}
      </div>
    </div>
  );
}

export function assertData(
  data: undefined | InsightsFetchResult,
): data is InsightsFetchResult {
  if (!data?.rows.length)
    throw new Error('Unexpectedly received empty data in ResultsTable.');
  return true;
}
