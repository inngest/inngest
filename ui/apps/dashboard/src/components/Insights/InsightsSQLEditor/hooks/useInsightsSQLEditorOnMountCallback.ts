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

    // TODO: This code is not currently running. It turns out that actually doing so would
    // require messy exterior code. This is here to demonstrate roughly the pattern that would
    // be needed if any of these "actions" truly needed to run disposable functions. As for now,
    // neither of them do anything truly global, so all necessary cleanup should happen just as
    // a result of the monaco editor unmounting.
    return () => {
      shortcutsDisposable.dispose();
      markersDisposable.dispose();
    };
  });

  return { onMount };
}
