'use client';

import { Button } from '@inngest/components/Button';

import { type InsightsResult, type InsightsState } from '../types';

interface ResultsTableFooterProps {
  data: InsightsResult;
  state: InsightsState;
}

export function ResultsTableFooter({ data, state }: ResultsTableFooterProps) {
  return (
    <div className="border-subtle border-t px-4 py-3">
      <div className="flex items-center justify-between">
        <span className="text-muted text-sm">
          {`${data.totalCount} ${data.totalCount === 1 ? 'row' : 'rows'}`}
        </span>

        {state === 'fetchingMore' && (
          <div className="flex flex-1 justify-center">
            <Button appearance="outlined" label="Loading more..." loading size="small" />
          </div>
        )}
      </div>
    </div>
  );
}
