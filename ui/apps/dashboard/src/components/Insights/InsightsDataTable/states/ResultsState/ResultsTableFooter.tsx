'use client';

import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button/Button';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ResultsTableFooter() {
  const { data, fetchMore, fetchMoreError, status } = useInsightsStateMachineContext();

  if (!['fetchingMore', 'fetchMoreError', 'success'].includes(status)) return null;
  if (!assertData(data)) return null;

  return (
    <div className="border-subtle flex h-[45px] items-center justify-between border-t py-0">
      {status === 'fetchMoreError' && (
        <Alert
          className="flex-1 rounded-none text-sm"
          inlineButton={
            <Button
              appearance="solid"
              className="ml-auto h-auto p-0 text-sm font-medium underline"
              kind="secondary"
              label="Retry"
              size="medium"
              onClick={() => {
                fetchMore();
              }}
            />
          }
          severity="error"
        >
          {fetchMoreError ?? FALLBACK_ERROR}
        </Alert>
      )}

      {(status === 'success' || status === 'fetchingMore') && (
        <div className="text-muted pl-3 text-sm">
          {`${data.totalCount} ${data.totalCount === 1 ? 'row' : 'rows'}`}
        </div>
      )}
    </div>
  );
}

export function assertData(data: undefined | InsightsFetchResult): data is InsightsFetchResult {
  if (!data?.entries.length) throw new Error('Unexpectedly received empty data in ResultsTable.');
  return true;
}
