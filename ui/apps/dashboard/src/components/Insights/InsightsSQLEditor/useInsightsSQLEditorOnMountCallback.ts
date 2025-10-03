'use client';

import type { MutableRefObject } from 'react';
import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { useLatest, useLatestCallback } from './useLatestCallback';
import { getCanRunQuery } from './utils';

// This hook makes use of the useLatest and useLatestCallback hooks to get around the fact that
// onMount will only run once when the Monaco editor initially mounts. Without this approach,
// the cmd+enter shortcut handler would see stale values.

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

function handleShortcuts(
  editor: Parameters<SQLEditorMountCallback>[0],
  monaco: Parameters<SQLEditorMountCallback>[1],
  latestQueryRef: MutableRefObject<string>,
  isRunningRef: MutableRefObject<boolean>,
  runQuery: () => void
) {
  return editor.onKeyDown((e) => {
    if (e.keyCode === monaco.KeyCode.Enter && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      e.stopPropagation();

      if (getCanRunQuery(latestQueryRef.current, isRunningRef.current)) runQuery();
    }
  });
}
