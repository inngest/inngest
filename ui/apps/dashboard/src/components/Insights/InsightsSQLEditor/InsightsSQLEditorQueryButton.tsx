'use client';

import { Button } from '@inngest/components/Button/Button';
import { RiPlayFill } from '@remixicon/react';
import { ulid } from 'ulid';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { useQueryHelperPanelContext } from '../QueryHelperPanel/QueryHelperPanelContext';

export function InsightsSQLEditorQueryButton() {
  const { isEmpty, query, runQuery, status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  const { addRecentQuery } = useQueryHelperPanelContext();

  return (
    <Button
      className="w-[110px]"
      disabled={isEmpty || isRunning}
      icon={<RiPlayFill className="h-4 w-4" />}
      iconSide="left"
      label={isRunning ? undefined : 'Run query'}
      loading={isRunning}
      onClick={async () => {
        const status = await runQuery();

        if (status === 'success') {
          addRecentQuery({ id: ulid(), text: query });
        }
      }}
      size="medium"
    />
  );
}
