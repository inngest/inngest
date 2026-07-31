import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import { lineColors } from '@/components/Metrics/utils';

// Red (lineColors[3]) is reserved for boolean `false` bars and must never be
// used for a numeric line chart (nor any near-red hue). Numeric scores cycle
// through this red-free subset of the metrics palette.
export const NUMERIC_LINE_COLORS = [
  lineColors[2], // blue
  lineColors[4], // purple
  lineColors[1], // green
  lineColors[0], // amber
];

// One stable color per score, assigned by position in the full available
// list so a score keeps its color regardless of which others are toggled
// off or filtered elsewhere. Cycles when there are more than 4 scores.
export function getScoreColors(
  scores: { name: string }[],
): Map<string, string> {
  const dark = isDark();
  const m = new Map<string, string>();
  scores.forEach((s, i) => {
    const [token, hex] = NUMERIC_LINE_COLORS[i % NUMERIC_LINE_COLORS.length];
    m.set(s.name, resolveColor(token, dark, hex));
  });
  return m;
}
