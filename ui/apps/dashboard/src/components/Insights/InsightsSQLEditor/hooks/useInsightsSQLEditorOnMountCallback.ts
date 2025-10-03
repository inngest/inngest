'use client';

import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';
import { handleShortcuts } from '../actions/handleShortcuts';
import { markTemplateVars } from '../actions/markTemplateVars';
import { useLatest, useLatestCallback } from './useLatestCallback';

type UseInsightsSQLEditorOnMountCallbackReturn = {
  onMount: SQLEditorMountCallback;
};

export function useInsightsSQLEditorOnMountCallback(): UseInsightsSQLEditorOnMountCallbackReturn {
  const { query, runQuery, status } = useInsightsStateMachineContext();

  const latestQueryRef = useLatest(query);
  const isRunningRef = useLatest(status === 'loading');

  const onMount: SQLEditorMountCallback = useLatestCallback((editor, monaco) => {
    const shortcutsDisposable = handleShortcuts(
      editor,
      monaco,
      latestQueryRef,
      isRunningRef,
      runQuery
    );

    const markersDisposable = markTemplateVars(editor, monaco);

    return () => {
      shortcutsDisposable.dispose();
      markersDisposable.dispose();
    };
  });

  return { onMount };
}
