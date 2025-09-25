'use client';

/**
 * Split utilities
 *
 * We distinguish between the "current" split and the "stored" split:
 * - Current split is applied directly to a CSS variable for instant, jank-free updates during drag.
 *   Reading/writing React state or localStorage on every mousemove would be too slow.
 * - Stored split is persisted (eg. in localStorage) when interaction ends so layouts are restored later.
 */
import type { Orientation } from './types';

export const SPLIT_CSS_VAR = '--split';

export type OnMoveOptions = {
  maxSplitPercentage: number;
  minSplitPercentage: number;
  orientation: Orientation;
};

export function computeSplitPercentageFromEvent(
  el: HTMLElement,
  ev: MouseEvent,
  orientation: Orientation
): number {
  const r = el.getBoundingClientRect();
  return orientation === 'vertical'
    ? ((ev.clientY - r.top) / r.height) * 100
    : ((ev.clientX - r.left) / r.width) * 100;
}

export function makeOnMove(el: HTMLElement, options: OnMoveOptions): (ev: PointerEvent) => void {
  const { maxSplitPercentage, minSplitPercentage, orientation } = options;

  return (ev: PointerEvent) => {
    const pct = computeSplitPercentageFromEvent(el, ev, orientation);
    writeCurrentSplit(el, clamp(pct, minSplitPercentage, maxSplitPercentage));
  };
}

type StopDragOptions = {
  maxSplitPercentage: number;
  minSplitPercentage: number;
  onMove: (ev: PointerEvent) => void;
  orientation: Orientation;
  splitKey?: string;
};

export function makeOnStopDrag(
  el: HTMLElement,
  { maxSplitPercentage, minSplitPercentage, onMove, splitKey }: StopDragOptions
) {
  return function onStop() {
    removeSplitListeners(onMove, onStop);

    if (splitKey) {
      const n = readCurrentSplit(el);
      if (n === null) return;

      writeStoredSplit(splitKey, clamp(n, minSplitPercentage, maxSplitPercentage));
    }
  };
}

export function addSplitListeners(
  onMove: (ev: PointerEvent) => void,
  onStop: (ev: Event) => void
): void {
  window.addEventListener('pointermove', onMove);
  window.addEventListener('pointerup', onStop);
  window.addEventListener('pointercancel', onStop);
  window.addEventListener('blur', onStop);
  window.addEventListener('contextmenu', onStop);
}

export function removeSplitListeners(
  onMove: (ev: PointerEvent) => void,
  onStop: (ev: Event) => void
): void {
  window.removeEventListener('pointermove', onMove);
  window.removeEventListener('pointerup', onStop);
  window.removeEventListener('pointercancel', onStop);
  window.removeEventListener('blur', onStop);
  window.removeEventListener('contextmenu', onStop);
}

export function readCurrentSplit(el: HTMLElement): number | null {
  const raw = el.style.getPropertyValue(SPLIT_CSS_VAR);
  const n = Number((raw || '').trim().replace('%', ''));
  return Number.isNaN(n) ? null : n;
}

export function readStoredSplit(splitKey: string): number | null {
  try {
    const stored = localStorage.getItem(splitKey);
    if (stored === null) return null;

    const n = Number(stored);
    return Number.isNaN(n) ? null : n;
  } catch {
    return null;
  }
}

export function writeCurrentSplit(el: HTMLElement, pct: number): void {
  el.style.setProperty(SPLIT_CSS_VAR, `${pct}%`);
}

export function writeStoredSplit(splitKey: string, value: number): void {
  try {
    localStorage.setItem(splitKey, String(value));
  } catch {
    console.warn('Failed to write stored resizeable split value.');
  }
}

export function initializeSplitFromStorage(
  el: HTMLElement,
  args: {
    defaultSplitPercentage: number;
    maxSplitPercentage: number;
    minSplitPercentage: number;
    splitKey?: string;
  }
) {
  const { defaultSplitPercentage, maxSplitPercentage, minSplitPercentage, splitKey } = args;

  let initial = defaultSplitPercentage;
  if (splitKey) {
    const stored = readStoredSplit(splitKey);
    if (stored !== null) initial = stored;
  }

  writeCurrentSplit(el, clamp(initial, minSplitPercentage, maxSplitPercentage));
}

function clamp(n: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, n));
}
