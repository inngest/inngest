'use client';

import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';

import { type InsightsResult, type InsightsState } from '../types';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

interface ResultsTableFooterProps {
  data: InsightsResult;
  fetchMoreError?: string;
  state: InsightsState;
}

export function ResultsTableFooter({ data, fetchMoreError, state }: ResultsTableFooterProps) {
  if (!['fetchingMore', 'fetchMoreError', 'success'].includes(state)) return null;

  return (
    <div className="border-subtle flex h-[45px] items-center justify-between border-t py-0">
      {state === 'fetchMoreError' && (
        <Alert className="flex-1 rounded-none text-sm" severity="error">
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
