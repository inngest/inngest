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

/**
 * What a row IS, which is the small extra fact the moments cannot carry.
 *
 * `started > ok` is compute on a step, the SDK reporting on a discovery
 * request, a child run on an invoke, elapsed sleep on a wait, and a userland
 * span on a span. Same pair of moments, five substances — so the row declares
 * its kind and the pair does the rest.
 *
 * `open` is the substance of a row that started and has not resolved: the
 * trailing bar when the last moment is `started`.
 */
export const KINDS = {
  step:  {ok:'good', failed:'bad', retry:'bad', cancelled:'stopped', open:'running'},
  disc:  {ok:'disc', failed:'disc!', retry:'disc!', cancelled:'stopped', open:'disc'},
  child: {ok:'child', failed:'child!', cancelled:'stopped', open:'child'},
  wait:  {ok:'waitok', timeout:'waitout', cancelled:'waitstop', open:'wait'},
  span:  {ok:'spanok', failed:'spanerr', done:'spanunset', open:'running'},
};

/**
 * The substance of the interval leading UP TO work starting. Which kind of
 * not-working it was is carried by the moment that opened it, not by the row.
 */
export const RESOLVED = new Set(['ok','failed','retry','timeout','cancelled','done']);

export const BEFORE_START = {
  queued:  'idle',      // waiting for the executor
  ok:      'idle',      // something resolved; the next thing is waiting
  failed:  'idle',
  planned: 'idle',      // a discovery request said this would run
  retry:   'backoff',   // the attempt threw; this is the wait before the next
  held:    'hold',      // concurrency, throttle, rate limit or debounce
};

/**
 * Moments to bars. `at` is [moment, position] in ascending position; `end` is
 * where the row stops when its last moment left it running.
 *
 * A `started` moment may name a third thing -- the kind of work it opens -- for
 * the row that changes substance partway: a step that then sleeps is one row,
 * one list of moments, and only that moment has to say so.
 *
 * This is the whole derivation. A figure declares what happened and the bars
 * follow, so changing what a substance means changes every figure at once
 * rather than every figure that remembered.
 */
export function derive(kind, at, end, opts={}){
  const K = KINDS[kind] || KINDS.step;
  // The head of a row can belong to the request that reported the step rather
  // than to the step: 'the short blue interval at the head of the next row'.
  // It is still one row of moments; only the first stretch of work is the
  // request's, so it takes the request's substances.
  const head = opts.reported ? KINDS.disc : null;
  let done = !head;                      // has the head request resolved yet?
  const segs = [];
  const table = () => done ? K : head;
  // Widths come out of a time axis, so 'ends exactly here' arrives as a
  // thousandth of a percent rather than zero. A bar that thin is not a bar.
  const push = (s, x, w) => { if(s && w > 1e-3) segs.push([s, x, w]); };

  for(let i=0; i<at.length-1; i++){
    const [a, x, as] = at[i], [b, x2] = at[i+1];
    if(RESOLVED.has(b)){
      push((KINDS[as] || table())[b], x, x2 - x);
      if(!done && b === 'ok') done = true; // the request reported; the step is next
    }else{
      push(BEFORE_START[a], x, x2 - x);
    }
  }
  const last = at[at.length-1];
  if(last && end != null && end > last[1]){
    // A row that is still in the state its last moment put it in.
    push(last[0] === 'started' ? (KINDS[last[2]] || table()).open : BEFORE_START[last[0]], last[1], end - last[1]);
  }
  return segs;
}

/** Substances that are work happening, as opposed to waiting for it. */
const WORKING = k => !['idle','backoff','hold'].includes(k.replace(/[!*]+$/,''));

/** The moment that OPENS an interval of not-working. */
const OPENS = {idle:'queued', backoff:'retry', hold:'held'};

/** The moment that CLOSES an interval of work. */
const CLOSES = {good:'ok', disc:'ok', child:'ok', spanok:'ok', waitok:'ok',
  bad:'failed', spanerr:'failed', spanunset:'done', waitout:'timeout',
  stopped:'cancelled', waitstop:'cancelled'};

/** Which KINDS table a substance belongs to. */
const TABLE_OF = {disc:'disc', child:'child', wait:'wait', waitok:'wait',
  waitout:'wait', waitstop:'wait', spanok:'span', spanerr:'span', spanunset:'span'};

/** Substances a row can sit in without ever resolving. */
const UNRESOLVED = new Set(['running','wait']);

/**
 * The moments a row of bars implies -- the inverse of `derive`, and the reading
 * that lets a figure written the older way still go through the rule.
 *
 * Every boundary is a moment. A bar of work ends by RESOLVING, even where the
 * drawing puts a single circle there and labels it `queued`: the step resolved
 * and the next thing started waiting, at the same instant.
 */
export function moments(segs){
  if(!segs.length) return {at:[], end:0, kind:'step', reported:false};
  const bare = k => k.replace(/[!*]+$/,'');
  const fail = k => /!\*?$/.test(k) || bare(k)==='bad';
  const at=[];
  const add=(n,p,s)=>{
    if(!n) return;
    const prev=at[at.length-1];
    if(prev && prev[0]===n && prev[1]===p) return;
    at.push(s?[n,p,s]:[n,p]);
  };
  const lastWork=segs.map(([k])=>WORKING(k)).lastIndexOf(true);
  segs.forEach(([k,x],i)=>{
    const prev=segs[i-1];
    // The previous work resolved here; another attempt still to come makes that
    // resolution a retry rather than the row's outcome.
    let retried=false;
    if(prev && WORKING(prev[0])){
      retried = fail(prev[0]) && i<=lastWork;
      add(retried ? 'retry' : CLOSES[bare(prev[0])], x);
    }
    // A retry already opens the interval that follows it, and it opens it as
    // backoff. Letting a `queued` in at the same instant would say the run was
    // waiting for the executor when it was waiting out a failure.
    if(!(retried && !WORKING(k)))
      add(WORKING(k) ? 'started' : OPENS[bare(k)], x, WORKING(k) ? (TABLE_OF[bare(k)]||'step') : null);
  });
  const last=segs[segs.length-1];
  if(WORKING(last[0]) && !UNRESOLVED.has(bare(last[0])))
    add(fail(last[0]) && bare(last[0])!=='bad' ? 'failed' : CLOSES[bare(last[0])], last[1]+last[2]);

  // What the row IS. A discovery bar followed by work of another substance is
  // the reported-at-the-head shape: the request is the head of the row and the
  // step's own work follows it.
  const work=segs.map(([k])=>k).filter(WORKING);
  const after=work.filter(k=>bare(k)!=='disc');
  const reported=work.some(k=>bare(k)==='disc') && after.length>0;
  const decisive=(reported?after:work)[0];
  const kind=decisive?(TABLE_OF[bare(decisive)]||'step'):'step';

  // A moment only names its substance where that is not the row's own, so the
  // ordinary row stays a plain pair.
  const trimmed=at.map((mo,j)=>
    mo.length===3 && (mo[2]===kind || (reported && j===0)) ? [mo[0],mo[1]] : mo);
  return {at:trimmed, end:last[1]+last[2], kind, reported};
}

/**
 * The grouping ribbon, derived.
 *
 * One discovery request can report several steps, and they are all queued at
 * the same instant -- the instant that request resolved. So a ribbon is not
 * something a figure places: it is every row whose step was queued at the same
 * moment, drawn at that moment. Two or more rows make a group; one does not,
 * which is why a lone step has no ribbon and never should.
 *
 * A row that carries its own reporting request at the head has TWO queue
 * moments: the request's and the step's. The step's is the one that matters,
 * and it is the first queue after the request resolved.
 */
export function ribbonGroups(rows){
  /**
   * Where the timing does not imply the relationship, the row states it: a
   * request row may name the rows it reported. Real spans carry that link
   * explicitly; identical queue instants are a property of figures, and a
   * figure showing members that queued a moment apart has to say what ties
   * them or the ribbon would be inferred away.
   */
  const named=[];
  rows.forEach((r,i)=>{
    if(!r.reports || !r.at) return;
    const k=r.at.findIndex(x=>RESOLVED.has(x[0]));
    if(k<0) return;
    const mem=[i].concat(r.reports
      .map(n=>rows.findIndex(o=>o.n===n))
      .filter(j=>j>=0));
    if(mem.length>1) named.push({x:+r.at[k][1].toFixed(3), rows:mem});
  });
  const claimed=new Set(named.flatMap(g=>g.rows));
  const marks=[];
  rows.forEach((r,i)=>{
    if(r.run || !r.at || !r.at.length) return;
    const m=r.at;
    let q;
    if(r.reported){
      const k=m.findIndex(x=>RESOLVED.has(x[0]));
      q = k<0 ? null : m.slice(k+1).find(x=>x[0]==='queued'||x[0]==='planned');
    }else{
      q = m.find(x=>x[0]==='queued'||x[0]==='planned');
    }
    if(q && !claimed.has(i)) marks.push([i, +q[1].toFixed(3)]);
  });
  const by=new Map();
  for(const [i,x] of marks){ if(!by.has(x)) by.set(x,[]); by.get(x).push(i); }
  return named.concat([...by.entries()]
      .filter(([,rs])=>rs.length>1)
      .map(([x,rs])=>({x, rows:rs})))
    .sort((a,b)=>a.x-b.x);
}

/**
 * The queue time a run starts with, which is worth a label and not a third of
 * the width.
 *
 * Many real runs sit enqueued far longer than they execute, and drawn to scale
 * that opens a gap at the head of every row before anything has happened. The
 * elastic rule does not catch it: the stretch is real queue time, and queue
 * time is exactly what the trace exists to show -- everywhere except here,
 * where it is the same fact on every row and says nothing about any of them.
 *
 * So the drawing starts when the run starts working, and the wait that came
 * first is named on the Run row instead.
 */
export function leadingQueue(rows){
  let first=Infinity;
  for(const r of rows){
    if(r.run) continue;
    for(const [k,x] of (r.segs||[])) if(WORKING(k)){ first=Math.min(first,x); break; }
  }
  return first===Infinity?0:first;
}

/** A duration, at the precision a label wants. */
export function human(ms){
  if(ms>=1000) return +(ms/1000).toFixed(ms>=10000?0:3)+'s';
  return Math.round(ms)+'ms';
}
