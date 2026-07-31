import { describe, expect, it } from 'vitest';

import { getScoreColors } from './colors';

describe('getScoreColors', () => {
  it('returns an empty map for no scores', () => {
    expect(getScoreColors([]).size).toBe(0);
  });

  it('assigns one color per score by position, cycling after 4', () => {
    const scores = [
      { name: 'a' },
      { name: 'b' },
      { name: 'c' },
      { name: 'd' },
      { name: 'e' },
    ];
    const colors = getScoreColors(scores);

    // In a node test environment `resolveColor` always falls back to its
    // hex default (no `window` global), so results are deterministic
    // regardless of light/dark theme.
    expect(colors.get('a')).toBe('#2389f1'); // blue
    expect(colors.get('b')).toBe('#6222df'); // purple
    expect(colors.get('c')).toBe('#2c9b63'); // green
    expect(colors.get('d')).toBe('#ec9923'); // amber
    expect(colors.get('e')).toBe('#2389f1'); // cycles back to blue
  });

  it("keeps a score's color stable based on its position in the list", () => {
    const colors = getScoreColors([{ name: 'x' }, { name: 'y' }]);
    expect(colors.get('x')).toBe('#2389f1'); // blue, position 0
    expect(colors.get('y')).toBe('#6222df'); // purple, position 1
  });
});
