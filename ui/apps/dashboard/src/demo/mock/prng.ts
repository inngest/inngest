/**
 * Deterministic pseudo-randomness for the demo mock server. Every value is
 * derived from a string/number seed so the demo renders identically on every
 * request — no `Math.random`. Keep this dependency-free and pure.
 */

/** mulberry32 — small, fast, seedable 32-bit PRNG. */
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** Stable 32-bit hash of a string (FNV-1a), used to derive seeds from ids. */
export function hashString(input: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    h ^= input.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** A small deterministic RNG bundle keyed by an arbitrary seed value. */
export class Rng {
  private next: () => number;

  constructor(seed: string | number) {
    this.next = mulberry32(typeof seed === 'number' ? seed : hashString(seed));
  }

  float(min = 0, max = 1): number {
    return min + (max - min) * this.next();
  }

  int(min: number, max: number): number {
    return Math.floor(this.float(min, max + 1));
  }

  bool(pTrue = 0.5): boolean {
    return this.next() < pTrue;
  }

  pick<T>(items: readonly T[]): T {
    return items[Math.floor(this.next() * items.length)];
  }

  /** Pick weighted: pairs of [value, weight]. */
  weighted<T>(pairs: readonly (readonly [T, number])[]): T {
    const total = pairs.reduce((sum, [, w]) => sum + w, 0);
    let r = this.next() * total;
    for (const [value, w] of pairs) {
      r -= w;
      if (r <= 0) return value;
    }
    return pairs[pairs.length - 1][0];
  }
}
