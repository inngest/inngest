'use client';

import { Banner } from '@inngest/components/Banner/Banner';
import { Button } from '@inngest/components/Button/Button';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ResultsTableFooter() {
  const { data, error, fetchMore, status } = useInsightsStateMachineContext();

  if (!['fetchingMore', 'fetchMoreError', 'success'].includes(status)) return null;
  if (!assertData(data)) return null;

  return (
    <div className="border-subtle flex h-[45px] items-center justify-between border-t py-0">
      {status === 'fetchMoreError' && (
        <Banner
          cta={
            <Button
              appearance="ghost"
              kind="danger"
              label="Retry"
              onClick={() => {
                fetchMore();
              }}
            />
          }
          severity="error"
        >
          {error?.message ?? FALLBACK_ERROR}
        </Banner>
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
