'use client';

import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';
import { handleShortcuts } from '../actions/handleShortcuts';
import { useLatest, useLatestCallback } from './useLatestCallback';

type UseInsightsSQLEditorOnMountCallbackReturn = {
  onMount: SQLEditorMountCallback;
};

export function useInsightsSQLEditorOnMountCallback(): UseInsightsSQLEditorOnMountCallbackReturn {
  const { query, runQuery, status } = useInsightsStateMachineContext();

  const latestQueryRef = useLatest(query);
  const isRunningRef = useLatest(status === 'loading');

  const onMount: SQLEditorMountCallback = useLatestCallback((editor, monaco) => {
    const disposable = handleShortcuts(editor, monaco, latestQueryRef, isRunningRef, runQuery);

    return () => {
      disposable.dispose();
    };
  });

  return { onMount };
}
