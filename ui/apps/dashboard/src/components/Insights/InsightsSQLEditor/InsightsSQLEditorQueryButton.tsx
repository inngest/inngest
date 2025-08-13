'use client';

import { Button } from '@inngest/components/Button/Button';
import { RiPlayFill } from '@remixicon/react';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';

export function InsightsSQLEditorQueryButton() {
  const { isEmpty, runQuery, status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  return (
    <Button
      className="w-[110px]"
      disabled={isEmpty || isRunning}
      icon={<RiPlayFill className="h-4 w-4" />}
      iconSide="left"
      label={isRunning ? undefined : 'Run query'}
      loading={isRunning}
      onClick={runQuery}
      size="medium"
    />
  );
}
