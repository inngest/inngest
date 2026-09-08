import React from 'react';
import { fig, resolveRow } from '../micro.mjs';
import { RESOLVED, setFeatures } from '../rules.mjs';

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
export function Trace({ rows, ms, lead, extra = '', under = '', label = '',
                        frame = true, running, breaks,
                        trim = true, compress = true, compressCompute = false,
                        multiple, ...opts }) {
  const html = React.useMemo(() => {
    // No guard on empty rows: a figure can be about the axis, and its content
    // is in `extra`. Bailing here dropped it from the page entirely.
    rows = rows || [];
    // What the design does FOR you, as properties of this drawing.
    setFeatures({ trim, compress, compressCompute, multiple });
    // Everything goes to fig(). It owns the whole path from events to drawing:
    // trimming the opening queue, the elastic axis, the bands, the frame. Doing
    // any of it here would be a second opinion about what the events mean.
    return fig(rows, extra, label, under, { frame, running, lead, ms, breaks, ...opts });
  }, [rows, ms, lead, extra, under, label, frame, running, breaks, trim, compress, compressCompute, multiple, opts]);

  // No wrapper class: it renders INTO the figure box the page already has.
  return React.createElement('div', {
    style: { display: 'contents' },
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
