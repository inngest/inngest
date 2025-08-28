'use client';

import { Button } from '@inngest/components/Button/Button';
import { RiCommandLine, RiCornerDownLeftFill } from '@remixicon/react';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';

function QueryButtonLabel({ isRunning }: { isRunning: boolean }) {
  if (isRunning) return null;

  return (
    <div className="flex items-center gap-2">
      <span>Run query</span>
      <div className="bg-primary-moderate flex shrink-0 gap-0.5 rounded-[4px] px-1 py-0.5">
        <RiCommandLine className="h-4 w-4" />
        <RiCornerDownLeftFill className="h-4 w-4" />
      </div>
    </div>
  );
}

export function InsightsSQLEditorQueryButton() {
  const { query, runQuery, status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  return (
    <Button
      className="w-[135px] font-medium"
      disabled={query.trim() === '' || isRunning}
      label={<QueryButtonLabel isRunning={isRunning} />}
      loading={isRunning}
      onClick={(e) => {
        runQuery();
        e.currentTarget.blur();
      }}
      size="medium"
    />
  );
}
