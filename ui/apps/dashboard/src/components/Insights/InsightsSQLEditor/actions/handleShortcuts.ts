'use client';

import { useEffect } from 'react';
import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

type EditorInstance = Parameters<SQLEditorMountCallback>[0];
type MonacoKeyEvent = Parameters<Parameters<EditorInstance['onKeyDown']>[0]>[0];

type ModKeyState = {
  altKey?: boolean;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
};

export type KeyCombo =
  | {
      alt?: boolean;
      // DOM: KeyboardEvent.code (e.g. 'KeyS', 'Enter')
      code: KeyboardEvent['code'];
      metaOrCtrl?: boolean;
      shift?: boolean;
    }
  | {
      alt?: boolean;
      // Monaco: numeric enum keyCode (e.g. monaco.KeyCode.KeyS)
      keyCode: number;
      metaOrCtrl?: boolean;
      shift?: boolean;
    };

export type ShortcutBinding = {
  combo: KeyCombo;
  handler: () => void;
};

type ModifiableEvent = {
  preventDefault(): void;
  stopPropagation(): void;
};

function doAction(e: ModifiableEvent, handler: () => void) {
  e.preventDefault();
  e.stopPropagation();
  handler();
}

function modsMatch(e: ModKeyState, c: KeyCombo): boolean {
  if (c.metaOrCtrl && !(e.metaKey || e.ctrlKey)) return false;
  if (c.alt && !e.altKey) return false;
  if (c.shift && !e.shiftKey) return false;
  return true;
}

export function bindEditorShortcuts(
  editor: EditorInstance,
  ...bindings: ReadonlyArray<ShortcutBinding>
) {
  return editor.onKeyDown((e: MonacoKeyEvent) => {
    for (const { combo, handler } of bindings) {
      if (!('keyCode' in combo)) continue;
      if (!modsMatch(e, combo)) continue;
      if (e.keyCode !== combo.keyCode) continue;
      return doAction(e, handler);
    }
  });
}

export function useDocumentShortcuts(...bindings: ReadonlyArray<ShortcutBinding>) {
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      for (const { combo, handler } of bindings) {
        if (!('code' in combo)) continue;
        if (!modsMatch(e, combo)) continue;
        if (e.code !== combo.code) continue;
        return doAction(e, handler);
      }
    }

    document.addEventListener('keydown', onKeyDown);

    return () => document.removeEventListener('keydown', onKeyDown);
  }, [bindings]);
}
