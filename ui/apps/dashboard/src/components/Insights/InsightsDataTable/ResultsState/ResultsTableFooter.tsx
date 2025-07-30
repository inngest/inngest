'use client';

import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button/Button';

import { useInsightsQueryContext } from '../../context';
import type { InsightsResult } from '../types';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ResultsTableFooter() {
  const { data, fetchMore, fetchMoreError, state } = useInsightsQueryContext();

  if (!['fetchingMore', 'fetchMoreError', 'success'].includes(state)) return null;
  if (!assertData(data)) return null;

  return (
    <div className="border-subtle flex h-[45px] items-center justify-between border-t py-0">
      {state === 'fetchMoreError' && (
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

      {(state === 'success' || state === 'fetchingMore') && (
        <div className="text-muted pl-3 text-sm">
          {`${data.totalCount} ${data.totalCount === 1 ? 'row' : 'rows'}`}
        </div>
      )}
    </div>
  );
}

export function assertData(data: undefined | InsightsResult): data is InsightsResult {
  if (!data?.entries.length) throw new Error('Unexpectedly received empty data in ResultsTable.');
  return true;
}
