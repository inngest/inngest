'use client';

import type { MutableRefObject } from 'react';
import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { getCanRunQuery } from '../utils';

export function handleShortcuts(
  editor: Parameters<SQLEditorMountCallback>[0],
  monaco: Parameters<SQLEditorMountCallback>[1],
  latestQueryRef: MutableRefObject<string>,
  isRunningRef: MutableRefObject<boolean>,
  runQuery: () => void
) {
  return editor.onKeyDown((e) => {
    if (hasKeys(e, ['metaOrCtrl', monaco.KeyCode.Enter])) {
      e.preventDefault();
      e.stopPropagation();
      if (getCanRunQuery(latestQueryRef.current, isRunningRef.current)) runQuery();
    }

    if (hasKeys(e, ['metaOrCtrl', monaco.KeyCode.KeyS])) {
      e.preventDefault();
      e.stopPropagation();
      console.log('Insights: Save query (Cmd/Ctrl+Alt+S)');
      return;
    }

    if (hasKeys(e, ['metaOrCtrl', 'alt', monaco.KeyCode.KeyT])) {
      e.preventDefault();
      e.stopPropagation();
      console.log('Insights: New tab (Cmd/Ctrl+Shift+N)');
      return;
    }
  });
}

type EditorInstance = Parameters<SQLEditorMountCallback>[0];
type KeyEvent = Parameters<Parameters<EditorInstance['onKeyDown']>[0]>[0];
type KeySpec = 'alt' | 'metaOrCtrl' | number;

function hasKeys(ev: KeyEvent, keys: ReadonlyArray<KeySpec>): boolean {
  for (const k of keys) {
    switch (k) {
      case 'alt':
        if (!ev.altKey) return false;
        break;
      case 'metaOrCtrl':
        if (!(ev.metaKey || ev.ctrlKey)) return false;
        break;
      default:
        if (ev.keyCode !== k) return false;
        break;
    }
  }

  return true;
}
