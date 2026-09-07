/**
 * THE RULES.
 *
 * Every decision the drawing makes, in one file. Change a number here and the
 * whole artifact moves — the figures, the fixtures, the docs page, all of it —
 * because nothing downstream holds a constant of its own.
 *
 * If you find yourself typing a number into `vocabulary.mjs`, `micro.mjs` or
 * `ds.mjs`, it belongs here instead. That is the whole point of the file: the
 * rules were spread across 2,142 lines of drawing code, so "change the rule"
 * meant "find every place that encoded it".
 */

/** The geometry of a row. Every renderer reads these rather than restating them. */
export const GEOM = {
  W:470, LBL:64, RGT:10, PLOT:396,   // PLOT = W - LBL - RGT
  ROW:17, TOP:6,
  BAR_H:7, RUN_H:8, TRACK_H:5,       // step bar, run profile slice, run track
  MARK_R:3, HALO:1.3,                // mark radius, and the ring of surface behind it
  MIN_W:1.4, MIN_FAIL_W:3.2,         // a bar never narrower than this on screen
  SPAN_ROW:9,                        // pitch for a userland span row
  SPAN_BAR:0.6,                      // span bar height, as a fraction of BAR_H
};

/**
 * When a stretch of dead time is worth compressing.
 *
 * Scale-free on purpose. A flat fraction of the run compresses ordinary queue
 * intervals — short, meaningful, and exactly what the trace exists to show. The
 * test that holds is against the WORK: a stretch qualifies when it is longer
 * than all the compute in the run put together, several times over.
 */
export const ELASTIC = {
  /** Dead must exceed this multiple of total compute. */
  computeMultiple: 3,
  /** ...and never less than this fraction of the run, however little work there is. */
  floor: 0.03,
  /** All bands together get at most this share of the plot. */
  budget: 16,
  /** A band is never narrower than this, nor wider. */
  minBand: 1.1,
  maxBand: 4,
  /** The cut takes the middle of a stretch, leaving this much of the bar either side. */
  inset: 2,
  /** Below these widths a band drops its label, then its tear. */
  labelAt: 2.6,
  tearAt: 1.8,
};

/** How a compressed band is drawn. */
export const BAND = {
  scrim: 0.28,                       // --ground over the band
  ruleOpacity: 0.9,                  // at one band...
  ruleFalloff: 0.09,                 // ...less this per extra band
  ruleFloor: 0.28,
  blur: 1.4,                         // stdDeviation at one band...
  blurFalloff: 0.16,                 // ...less this per extra band
  blurFloor: 0.3,
  desaturate: 0.55,
  tearZigs: [2, 3],                  // min, max oscillations
  tearAmp: 1.9,
  tearWidth: 1.3,
};

/** The surround: the Run row above, finalization below. */
export const FRAME = {
  opacity: 0.34,
  blur: 0.62,
  desaturate: 0.25,
  finQueue: 2.2,                     // finalization waits, then runs
  finRun: 4.5,
};

/** Attention. */
export const FOCUS = {
  dim: 0.15,                         // everything off the path
  selBand: 0.13,                     // the selected row's wash
  selInset: 2,                       // ...less this, so two selections do not fuse
  ribbonDim: 0.35,
};

/** What the panel can move, and where each starts. */
export const CONTROLS = [
  {k:'--geo-bar',  n:'step bar height',   d:GEOM.BAR_H,   lo:0, hi:16},
  {k:'--geo-row',  n:'row pitch',         d:GEOM.ROW,     lo:0, hi:40},
  {k:'--geo-mark', n:'event radius',      d:GEOM.MARK_R,  lo:0, hi:8,  st:0.25},
  {k:'--geo-sbar', n:'span bar height',   d:+(GEOM.BAR_H*GEOM.SPAN_BAR).toFixed(1), lo:0, hi:12, st:0.2},
  {k:'--geo-span', n:'span row pitch',    d:GEOM.SPAN_ROW,lo:0, hi:24},
  {k:'--geo-run',  n:'run row height',    d:GEOM.RUN_H,   lo:0, hi:18},
  {k:'--geo-trk',  n:'run track height',  d:GEOM.TRACK_H, lo:0, hi:14, st:0.5},
  {k:'--geo-gap',  n:'gap between segments', d:0,         lo:0, hi:8,  st:0.25},
];

/**
 * What a bar is made of, and what an interval between two moments is.
 *
 * This is the derivation the artifact's own Concepts page describes: a row is a
 * list of moments, and the interval between two of them is a bar whose kind
 * follows from the pair. Authoring bars directly is the older way round.
 */
export const BETWEEN = {
  'queued>started':  'idle',
  'queued>planned':  'idle',
  'planned>started': 'idle',
  'started>ok':      'good',
  'started>failed':  'bad',
  'started>retry':   'bad',
  'retry>started':   'backoff',
  'started>timeout': 'waitout',
  'started>done':    'spanunset',
  'started>cancelled':'stopped',
};

/** The interval a pair of moments implies, or null if the pair says nothing. */
export const between = (a, b) => BETWEEN[`${a}>${b}`] || null;
