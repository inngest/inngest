import { useEffect, useRef } from 'react';
import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

type EditorInstance = Parameters<SQLEditorMountCallback>[0];
type MonacoKeyEvent = Parameters<Parameters<EditorInstance['onKeyDown']>[0]>[0];

type ModKeyState = {
  altKey?: boolean;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
};

type NormalizedKeyEvent = ModKeyState & {
  code?: string;
  keyCode?: number;
};

type DomKeyCombo = {
  alt?: boolean;
  code: KeyboardEvent['code'];
  keyCode?: never; // ensure mutual exclusivity at the type level
  metaOrCtrl?: boolean;
  shift?: boolean;
};

type MonacoKeyCombo = {
  alt?: boolean;
  code?: never; // ensure mutual exclusivity at the type level
  keyCode: number;
  metaOrCtrl?: boolean;
  shift?: boolean;
};

export type KeyCombo = DomKeyCombo | MonacoKeyCombo;

export type ShortcutBinding = {
  combo: KeyCombo;
  handler: () => void;
};

type ModifiableEvent = {
  preventDefault: () => void;
  stopPropagation: () => void;
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

function findMatchingHandler(
  bindings: ReadonlyArray<ShortcutBinding>,
  ev: NormalizedKeyEvent,
): (() => void) | undefined {
  for (const { combo, handler } of bindings) {
    if (!modsMatch(ev, combo)) continue;

    if ('code' in combo) {
      if (ev.code !== combo.code) continue;
      return handler;
    }

    if ('keyCode' in combo) {
      if (ev.keyCode !== combo.keyCode) continue;
      return handler;
    }
  }

  return undefined;
}

export function bindEditorShortcuts(
  editor: EditorInstance,
  bindings: ReadonlyArray<ShortcutBinding>,
) {
  return editor.onKeyDown((e: MonacoKeyEvent) => {
    const handler = findMatchingHandler(bindings, e);
    if (handler !== undefined) return doAction(e, handler);
  });
}

export function useDocumentShortcuts(bindings: ReadonlyArray<ShortcutBinding>) {
  const latestBindingsRef = useRef<ReadonlyArray<ShortcutBinding>>(bindings);

  useEffect(() => {
    latestBindingsRef.current = bindings;
  }, [bindings]);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const handler = findMatchingHandler(latestBindingsRef.current, e);
      if (handler !== undefined) return doAction(e, handler);
    }

    document.addEventListener('keydown', onKeyDown);

    return () => document.removeEventListener('keydown', onKeyDown);
  }, []);
}
