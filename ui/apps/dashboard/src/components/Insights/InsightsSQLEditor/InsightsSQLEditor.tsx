'use client';

import { useCallback, useLayoutEffect, useRef } from 'react';
import { SQLEditor, type SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { getCanRunQuery } from './utils';

export function InsightsSQLEditor() {
  const { onChange, query, runQuery, status } = useInsightsStateMachineContext();

  const latestQueryRef = useLatest(query);
  const isRunningRef = useLatest(status === 'loading');

  // useLatestCallback ensures Monaco's onMount gets a stable reference while the callback
  // nevertheless executes with fresh state (query, isRunning, runQuery).
  const handleEditorMount: SQLEditorMountCallback = useLatestCallback((editor, monaco) => {
    const disposable = editor.onKeyDown((e) => {
      if (e.keyCode === monaco.KeyCode.Enter && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        e.stopPropagation();

        if (getCanRunQuery(latestQueryRef.current, isRunningRef.current)) {
          runQuery();
        }
      }
    });

    return () => {
      disposable.dispose();
    };
  });

  return <SQLEditor content={query} onChange={onChange} onMount={handleEditorMount} />;
}

function useLatest<T>(value: T) {
  const r = useRef(value);

  useLayoutEffect(() => {
    r.current = value;
  }, [value]);

  return r;
}

function useLatestCallback<A extends unknown[], R>(cb: (...args: A) => R) {
  const latest = useLatest(cb);

  return useCallback((...args: A) => {
    return latest.current(...args);
  }, []);
}
