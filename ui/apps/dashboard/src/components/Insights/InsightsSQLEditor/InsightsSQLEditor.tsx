'use client';

import { useCallback, useLayoutEffect, useRef } from 'react';
import { SQLEditor, type SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { getCanRunQuery } from './utils';

export function InsightsSQLEditor() {
  const { onChange, query, runQuery, status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  // useLatestCallback ensures Monaco's onMount gets a stable reference while the callback
  // nevertheless executes with fresh state (query, isRunning, runQuery).
  const handleEditorMount: SQLEditorMountCallback = useLatestCallback((editor, monaco) => {
    const disposable = editor.onKeyDown((e) => {
      if (e.keyCode === monaco.KeyCode.Enter && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        e.stopPropagation();

        if (getCanRunQuery(query, isRunning)) runQuery();
      }
    });

    return () => {
      disposable.dispose();
    };
  });

  return <SQLEditor content={query} onChange={onChange} onMount={handleEditorMount} />;
}

function useLatestCallback<A extends any[], R>(callback: (...args: A) => R) {
  const ref = useRef(callback);

  useLayoutEffect(() => {
    ref.current = callback;
  }, [callback]);

  return useCallback((...args: A) => {
    return ref.current(...args);
  }, []);
}
