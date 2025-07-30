'use client';

import { Button } from '@inngest/components/Button/Button';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { RiPlayFill } from '@remixicon/react';

import { useInsightsQueryContext } from '../context';

export function InsightsSQLEditorQueryButton() {
  const { isEmpty, runQuery, state } = useInsightsQueryContext();
  const isRunning = state === 'loading';

  return (
    <Button
      className="w-[110px]"
      disabled={isEmpty || isRunning}
      icon={isRunning ? <IconSpinner className="fill-white" /> : <RiPlayFill className="h-4 w-4" />}
      iconSide="left"
      label={isRunning ? undefined : 'Run query'}
      onClick={runQuery}
      size="medium"
    />
  );
}
