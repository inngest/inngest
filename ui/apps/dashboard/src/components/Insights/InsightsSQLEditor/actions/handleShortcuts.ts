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
    if (e.keyCode === monaco.KeyCode.Enter && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      e.stopPropagation();

      if (getCanRunQuery(latestQueryRef.current, isRunningRef.current)) runQuery();
    }
  });
}
