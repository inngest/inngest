import React from 'react';
import { fig, layout, resolveRow } from '../micro.mjs';
import { RESOLVED } from '../rules.mjs';

/**
 * The trace renderer, as a component.
 *
 * ONE renderer. It takes the events a run is made of and nothing else: rows of
 * moments, in milliseconds. Everything the drawing shows follows from them —
 * the bars between moments, the marks on them, the ribbons over steps one
 * request planned, the cables into what caused a row, the elastic axis, the
 * Run row. Concepts, Scenarios, Fixtures and Docs all draw through this.
 *
 * It renders by calling the same pure functions the build calls, which is the
 * point: there is one implementation and this is its face. Wrapping rather than
 * reimplementing is deliberate — a second copy in JSX would be a second
 * renderer with a nicer syntax.
 */
export function Trace({ rows, ms, lead, at, frame = true, running, breaks, ...opts }) {
  const html = React.useMemo(() => {
    if (!rows || !rows.length) return '';
    // Rows measured in real time go through the elastic rule on the way in;
    // rows given as proportions are already on the figure's axis.
    if (ms) {
      const L = layout(ms, rows, { plot: 100 });
      return fig(L.rows, '', '', '', { frame, running, lead, breaks: L.breaks, ...opts });
    }
    return fig(rows, '', '', '', { frame, running, lead, breaks, ...opts });
  }, [rows, ms, lead, frame, running, breaks, opts]);

  return React.createElement('div', {
    className: 'figure',
    dangerouslySetInnerHTML: { __html: html },
  });
}

/**
 * A run as it stood at a moment: every event up to it, and a row that had
 * started and not resolved is still going.
 */
export function clipTo(rows, t) {
  return rows
    .map((r) => {
      const at = r.at.filter((m) => m[1] <= t + 1e-6);
      if (!at.length) return null;
      const open = !RESOLVED.has(at[at.length - 1][0]);
      return { ...r, at, end: open ? t : r.end };
    })
    .filter(Boolean);
}

export { resolveRow };
