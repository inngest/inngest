'use client';

import { useCallback, useLayoutEffect, useRef } from 'react';
import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
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
    const disposable = editor.onKeyDown((e) => {
      if (e.keyCode === monaco.KeyCode.Enter && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        e.stopPropagation();

        if (getCanRunQuery(latestQueryRef.current, isRunningRef.current)) runQuery();
      }
    });

    return () => {
      disposable.dispose();
    };
  });

  return { onMount };
}

// Uses a ref to ensure that the latest value is always available.
function useLatest<T>(value: T) {
  const r = useRef(value);

  useLayoutEffect(() => {
    r.current = value;
  }, [value]);

  return r;
}

// Extends useLatest to generate an always up-to-date callback.
function useLatestCallback<A extends unknown[], R>(cb: (...args: A) => R) {
  const latest = useLatest(cb);

  return useCallback((...args: A) => {
    return latest.current(...args);
  }, []);
}
