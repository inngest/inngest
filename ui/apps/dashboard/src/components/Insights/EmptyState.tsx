'use client';

import { Button } from '@inngest/components/Button';
import { RiExternalLinkLine, RiTableView } from '@remixicon/react';

type EmptyStateProps = {
  onSeeExamples?: () => void;
};

export function EmptyState({ onSeeExamples }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-12">
      <div className="bg-canvasSubtle flex h-[56px] w-[56px] items-center justify-center rounded-lg p-3">
        <RiTableView className="text-muted h-6 w-6" />
      </div>
      <div className="flex flex-col gap-2">
        <h3 className="text-basis text-xl font-medium">Your query results will appear here</h3>
        <p className="text-muted text-sm">Run a query to start generating insights.</p>
      </div>
      {onSeeExamples && (
        <Button
          appearance="outlined"
          className="text-md"
          icon={<RiExternalLinkLine />}
          iconSide="left"
          kind="primary"
          label="See examples"
          onClick={onSeeExamples}
          size="medium"
        />
      )}
    </div>
  );
}
