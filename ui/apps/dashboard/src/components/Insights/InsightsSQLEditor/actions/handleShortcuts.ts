'use client';

import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

type ShortcutActions = {
  onNewTab: () => void;
  onRun: () => void;
  onSave: () => void;
};

export function handleShortcuts(
  editor: Parameters<SQLEditorMountCallback>[0],
  monaco: Parameters<SQLEditorMountCallback>[1],
  actions: ShortcutActions
) {
  return editor.onKeyDown((e) => {
    if (hasKeys(e, ['metaOrCtrl', monaco.KeyCode.Enter])) {
      doAction(e, actions.onRun);
    } else if (hasKeys(e, ['metaOrCtrl', 'alt', monaco.KeyCode.KeyS])) {
      doAction(e, actions.onSave);
    } else if (hasKeys(e, ['metaOrCtrl', 'alt', monaco.KeyCode.KeyT])) {
      doAction(e, actions.onNewTab);
    }
  });
}

function doAction(e: KeyEvent, action: () => void) {
  e.preventDefault();
  e.stopPropagation();
  action();
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
