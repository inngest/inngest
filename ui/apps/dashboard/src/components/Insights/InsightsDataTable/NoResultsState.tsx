'use client';

import { Button } from '@inngest/components/Button';
import { RiExternalLinkLine, RiTableView } from '@remixicon/react';

import { useInsightsQueryContext } from '../context';

export function NoResultsState() {
  const { seeExamples } = useInsightsQueryContext();

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4">
      <div className="flex max-w-[410px] flex-col items-center gap-4">
        <div className="bg-canvasSubtle flex h-[56px] w-[56px] items-center justify-center rounded-lg p-3">
          <RiTableView className="text-light h-6 w-6" />
        </div>
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-basis text-xl font-medium">No results found</h3>
          <p className="text-muted text-sm">
            We couldn't find any results matching your search. Try adjusting your query or browse
            our examples for inspiration.
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
