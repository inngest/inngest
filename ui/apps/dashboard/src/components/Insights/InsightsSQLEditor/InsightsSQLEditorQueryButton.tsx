'use client';

import { useEffect } from 'react';
import { Button } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiCommandLine, RiCornerDownLeftFill } from '@remixicon/react';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { getCanRunQuery } from './utils';

function QueryButtonLabel({ disabled, isRunning }: { disabled: boolean; isRunning: boolean }) {
  if (isRunning) return null;

  return (
    <div className="flex items-center gap-2">
      <span>Run query</span>
      <div
        className={cn(
          'flex shrink-0 gap-0.5 rounded-[4px] px-1 py-0.5',
          disabled ? 'bg-muted' : 'bg-primary-moderate'
        )}
      >
        <RiCommandLine className="h-4 w-4" />
        <RiCornerDownLeftFill className="h-4 w-4" />
      </div>
    </div>
  );
}

export function InsightsSQLEditorQueryButton() {
  const { query, runQuery, status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';
  const canRunQuery = getCanRunQuery(query, isRunning);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Enter' && (event.metaKey || event.ctrlKey) && canRunQuery) {
        event.preventDefault();
        event.stopPropagation();
        runQuery();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [canRunQuery, runQuery]);

  return (
    <Button
      className="w-[135px] font-medium"
      disabled={!canRunQuery}
      label={<QueryButtonLabel isRunning={isRunning} disabled={!canRunQuery} />}
      loading={isRunning}
      onClick={(e) => {
        runQuery();
        e.currentTarget.blur();
      }}
      size="medium"
    />
  );
}
