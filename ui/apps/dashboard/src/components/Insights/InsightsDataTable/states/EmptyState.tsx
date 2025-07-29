'use client';

import { Button } from '@inngest/components/Button';
import { RiExternalLinkLine, RiTableView } from '@remixicon/react';

import { useInsightsQueryContext } from '../../context';

export function EmptyState() {
  const { seeExamples } = useInsightsQueryContext();

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4">
      <div className="flex max-w-[410px] flex-col items-center gap-4">
        <div className="bg-canvasSubtle flex h-[56px] w-[56px] items-center justify-center rounded-lg p-3">
          <RiTableView className="text-light h-6 w-6" />
        </div>
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-basis text-xl font-medium">Your query results will appear here</h3>
          <p className="text-muted text-sm">
            Run a query to analyze your data and the results will be displayed here. If you need a
            starting point, check out our examples.
          </p>
        </div>
        <Button
          appearance="outlined"
          icon={<RiExternalLinkLine />}
          iconSide="left"
          kind="primary"
          label="See examples"
          onClick={seeExamples}
          size="medium"
        />
      </div>
    </div>
  );
}
