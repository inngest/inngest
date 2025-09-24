export function clamp(n: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, n));
}

const SPLIT_VAR = '--split';
export function updateSplit(el: HTMLElement, pct: number): void {
  el.style.setProperty(SPLIT_VAR, `${pct}%`);
}
