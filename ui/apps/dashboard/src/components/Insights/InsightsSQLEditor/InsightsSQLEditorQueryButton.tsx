'use client';

import { Button } from '@inngest/components/Button/Button';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { KeyboardShortcut } from '../KeyboardShortcut';
import { useDocumentShortcuts } from './actions/handleShortcuts';
import { getCanRunQuery } from './utils';

function QueryButtonLabel({ disabled, isRunning }: { disabled: boolean; isRunning: boolean }) {
  if (isRunning) return null;

  return (
    <div className="flex items-center gap-2">
      <span>Run query</span>
      <KeyboardShortcut
        backgroundColor={disabled ? 'bg-muted' : 'bg-primary-moderate'}
        keys={['cmd', 'ctrl', 'enter']}
      />
    </div>
  );
}

export function InsightsSQLEditorQueryButton() {
  const { query, runQuery, status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';
  const canRunQuery = getCanRunQuery(query, isRunning);

  useDocumentShortcuts([
    {
      combo: { code: 'Enter', metaOrCtrl: true },
      handler: () => {
        if (canRunQuery) runQuery();
      },
    },
  ]);

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
