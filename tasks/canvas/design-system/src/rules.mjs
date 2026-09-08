/**
 * Features that change WHAT IS DRAWN, switchable at build time.
 *
 * The panel offers them as toggles, and the page carries a variant of every
 * figure they change. They cannot be CSS like the spacing sliders: hiding a
 * run's opening queue re-lays the whole trace, and compressing dead time
 * changes where every bar sits.
 */
export const ENV = typeof process !== 'undefined' && process.env ? process.env : {};
export const FEAT = {
  trim:     ENV.DS_NOTRIM     !== '1',
  compress: ENV.DS_NOCOMPRESS !== '1',
};

/**
 * The features are PROPERTIES of a drawing, not of the process drawing it.
 *
 * At build time they come from the environment; in the page they come from the
 * component's props, so switching one re-renders rather than swapping between
 * four pre-built copies of every figure. Set immediately before rendering and
 * read synchronously inside it.
 */
export function setFeatures(f){
  if(f && 'trim' in f) FEAT.trim=!!f.trim;
  if(f && 'compress' in f) FEAT.compress=!!f.compress;
}

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
  MAX_SCALE: 3,                      // the most a hand-placed figure is scaled up to fill the width
  W:470, LBL:64, RGT:10, PLOT:396,   // PLOT = W - LBL - RGT
  ROW:17, TOP:6,
  BAR_H:7, RUN_H:8, TRACK_H:5,       // step bar, run profile slice, run track
  MARK_R:3, HALO:1.3,                // mark radius, and the ring of surface behind it
  MIN_W:1.4, MIN_FAIL_W:3.2,         // a bar never narrower than this on screen
  ROW_TOP: 9,                        // where the first row sits under the top edge
  SPAN_EXIT: 4,                      // the extra gap coming back out of a block of spans
  SPAN_ROW:9,                        // pitch for a userland span row
  SPAN_BAR:0.6,                      // span bar height, as a fraction of BAR_H
  /**
   * The label column, fixed so that every figure begins its bars in the same
   * place. A divider sits at its right edge, and a name that would cross the
   * divider is cut rather than allowed to run on under the trace.
   */
  LBL_X:6,                           // where a row label starts
  LBL_GLYPH:3,                       // ...and the width of the "collapsed rows" mark left of it
  LBL_INDENT:5,                      // a userland span sits in under its step
  LBL_RULE:8,                        // the divider, this far left of the plot
                                     // — clear of the mark that sits on the plot's first pixel
  LBL_PAD:3,                         // clearance kept between a label and the divider
  LBL_FONT:5.5, LBL_SPAN_FONT:5,     // row label size, and the smaller one a span gets
  LBL_CH:0.6,                        // monospace advance, as a fraction of the font size
  /**
   * The ring that says a moment arrived by CHECKPOINT.
   *
   * A third channel, because the two a mark already has are taken: fill says
   * whether the row has resolved, colour says what happened. How we came to
   * know a moment is orthogonal to both and can land on any of them, so it
   * cannot borrow either. A concentric ring is purely additive -- it composes
   * with every colour and both fills, and implies nothing about good or bad.
   */
  CP_GAP:1.9,                        // how far outside the mark the ring sits
  CP_W:0.8,                          // ...and how heavy it is
  CP_O:0.55,                         // ...and how loud
};

/**
 * A name cut to fit the label column.
 *
 * The column is fixed, so something has to give when a name is longer than it
 * is wide: either the name runs on into the trace or it is elided. SVG has no
 * `text-overflow`, so the cut is made here -- which is arithmetic and not
 * measurement only because the labels are monospace and every glyph is the
 * same width.
 *
 * The full name stays in the markup as `data-n`, so nothing reading a figure
 * back -- the code/figure check, a tooltip -- ever sees the shortened form.
 */
export function elide(text, width, font){
  const max=Math.floor(width/(font*GEOM.LBL_CH));
  if(max<1) return "";
  return text.length<=max ? text : text.slice(0,max-1)+"\u2026";
}

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
  blurFalloff: 0.16,                 // ...less this per extra band
  blurFloor: 0.3,
  tearZigs: [2, 3],                  // min, max oscillations
  tearAmp: 1.9,
  tearWidth: 1.3,
};

/** The surround: the Run row above, finalization below. */
export const FRAME = {
  /**
   * The finalization the frame supplies, for a run that did not record one.
   *
   * FRACTIONS OF THE RUN, and only a fallback. What the frame would rather say
   * is what this run's own requests did -- the typical time one spent queued
   * and the typical time one spent executing -- and it only reaches for these
   * when the run contains no request to copy. A fixed number of milliseconds
   * here would be a claim about every run ever drawn.
   *
   * These used to be percentages of the AXIS, alongside a `rowsAxis: 86` that
   * reserved the last 14% of the width for them. Two unrelated numbers for one
   * piece of space: the rows were laid across 86 on the assumption the frame
   * would fill the rest, and the frame filled whatever its medians came to.
   * Where they disagreed -- which was nearly every figure -- the trace stopped
   * short of its own axis. The finalization is a row of moments on the run's
   * timeline now, so there is one number and it is the run.
   */
  finQueue: 0.022,
  finRun: 0.045,
};

/**
 * The tick strip: what the width is worth, in time.
 *
 * A compressed axis makes a short interval look long -- eleven milliseconds of
 * SDK execution takes half the width once two seconds of sleep have collapsed
 * to a band -- and nothing on the drawing said so. The ticks are placed at even
 * DRAWN positions and labelled with the time each one falls on, which is the
 * way round that works on an axis that is not linear: even spacing in time
 * would pile every tick into the live stretches.
 */
export const AXIS = {
  ticks: 5,                          // including both ends
  strip: 20,                         // the height it takes above the figure
  tick: 3,                           // the length of a tick mark
  line: 6,                           // ...and the gap between the two label lines
  font: 5.5,
  /**
   * Ticks are spaced within each STRETCH of the axis, not across the whole of
   * it. A compressed band is a discontinuity -- the time either side of it is
   * not the same distance apart as the pixels say -- so a tick placed across
   * one measures nothing, and the readings that matter most are the ones at
   * its edges: what the clock said going in, and what it said coming out.
   */
  spacing: 25,                       // drawn units between ticks within a stretch
  minSeg: 9,                         // ...below which a stretch gets a single one
};

/**
 * A tick's label, split across two lines.
 *
 * One line cannot carry both. `human()` rounds to a single unit, so on a run
 * measured in seconds two adjacent ticks both read "2s" and the axis stops
 * being able to tell them apart -- which is the whole reason the strip exists.
 * So the seconds and above go on top, the milliseconds below, and the bottom
 * line is always there: it is the resolution the timestamps actually have.
 *
 * Zero-padded when there is a line above it, because then it is a fraction of
 * that number rather than a quantity of its own -- "2s / 067ms" reads as one
 * time, "2s / 67ms" reads as two.
 */
export function tickLabel(ms){
  const t=Math.max(0, Math.round(ms));
  const rem=t%1000, secs=Math.floor(t/1000);
  if(!secs) return ['', rem+'ms'];
  const d=Math.floor(secs/86400), h=Math.floor(secs%86400/3600),
        m=Math.floor(secs%3600/60), s=secs%60;
  const parts=[];
  if(d) parts.push(d+'d');
  if(h) parts.push(h+'h');
  if(m) parts.push(m+'m');
  if(s||!parts.length) parts.push(s+'s');
  return [parts.join(' '), String(rem).padStart(3,'0')+'ms'];
}

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
      q = k<0 ? null : m.slice(k+1).find(x=>x[0]==='planned');
    }else{
      q = m.find(x=>x[0]==='planned');
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

/**
 * A duration, at the precision a label wants.
 *
 * Rounded on purpose. These are measured: a sleep asked for 2s and took
 * 1.996s, and a band that reads "1.996s" is reporting the measurement error
 * rather than the sleep. One or two significant figures is what a label on a
 * compressed band is for -- the exact number lives on the row.
 */
export function human(ms){
  const s=ms/1000;
  if(ms<1000)    return Math.round(ms)+'ms';
  if(s<90)       return +s.toFixed(s<10?1:0)+'s';
  const m=s/60;
  if(m<90)       return Math.round(m)+'m';
  const h=m/60;
  if(h<36)       return +h.toFixed(h<10?1:0)+'h';
  return +(h/24).toFixed(h/24<10?1:0)+'d';
}

/**
 * The instant the REQUEST at the head of a row was queued -- which is the
 * moment some earlier work finished and caused it. A cable ends here.
 */
export function requestQueuedAt(r){
  if(r.run || !r.at || !r.at.length) return null;
  const q=r.at.find(x=>x[0]==='queued'||x[0]==='planned');
  return q ? +q[1].toFixed(3) : null;
}

/** The instant a row's own step was queued, or null if it never waited. */
export function queuedAt(r){
  if(r.run || !r.at || !r.at.length) return null;
  const m=r.at;
  let q;
  if(r.reported){
    const k=m.findIndex(x=>RESOLVED.has(x[0]));
    q = k<0 ? null : m.slice(k+1).find(x=>x[0]==='queued'||x[0]==='planned');
  }else{
    q = m.find(x=>x[0]==='queued'||x[0]==='planned');
  }
  return q ? +q[1].toFixed(3) : null;
}

/** The instant a row finished -- its own outcome, not its request's. */
export function resolvedAt(r){
  if(r.run || !r.at || !r.at.length) return null;
  for(let i=r.at.length-1;i>=0;i--) if(RESOLVED.has(r.at[i][0])) return +r.at[i][1].toFixed(3);
  return null;
}

/**
 * The cables into one row: what had to finish before the request that produced
 * it could be queued.
 *
 * A step is queued because a discovery request said so, and that request was
 * queued because some work finished. So the causes of a row queued at Q are the
 * rows that RESOLVED in the window since the last request went out -- which is
 * why a Promise.all draws three cables into the step after it and a race draws
 * one, without either figure being told which.
 *
 * A row queued alongside others is reached by the ribbon instead; a ribbon
 * already says "one request produced all of these", and a cable to each would
 * be the same fact drawn twice.
 */
export function cables(rows, target){
  const t=rows[target];
  if(!t) return [];
  // Where the request was queued, not where the step was: on a row that
  // carries its own request those are different moments, and it is the request
  // that the earlier work caused.
  const q=requestQueuedAt(t);
  if(q==null) return [];
  const others=rows.map((r,i)=>i===target?null:requestQueuedAt(r)).filter(v=>v!=null&&v<q);
  const prev=others.length?Math.max(...others):-Infinity;
  return rows.map((r,i)=>{
    if(i===target) return null;
    const e=resolvedAt(r);
    return (e!=null && e>prev && e<=q) ? {from:i, x:e, to:target, q} : null;
  }).filter(Boolean);
}

/**
 * The mark each moment draws.
 *
 * Marks used to be read back off the bars by `autoDots`, which had to guess:
 * it called a row's first mark `queued` whatever it was, so a row that opens
 * already running -- a captured fixture whose leading queue has been trimmed
 * away -- still drew the queue mark the trim exists to remove.
 *
 * A moment knows what it is. `held` draws the queue mark because being held is
 * still waiting; the bar it opens is what says the wait had a reason.
 */
export const MARK_OF = {
  queued:'queued', held:'queued', planned:'ribbon', started:'hollow',
  retry:'hollow-bad', ok:'ok', failed:'failed', timeout:'timeout',
  cancelled:'cancelled', done:'done',
};

/** The marks a row draws, from its moments. */
export function marks(at){
  const out=[];
  for(const [n,x,sub] of at||[]){
    const c=MARK_OF[n];
    if(!c) continue;
    // Two moments at one instant -- a request resolving as the next step is
    // queued -- are one circle, and it is the later one that names it.
    //
    // Except at the row's OPENING, where the first moment names it instead.
    // The rule is about a transition: one state ending as the next begins, and
    // the circle carries what it became. A row has no earlier state, so there
    // is no transition and nothing for the second moment to be the result of.
    // Dropping the first mark there deleted the queue mark from every step the
    // SDK reported and ran in one request -- the step was queued, at the same
    // instant it started, and the row was drawn as though it never had been.
    if(out.length && Math.abs(out[out.length-1].p-x)<1e-9){
      if(out.length===1) continue;
      out.pop();
    }
    // A mark says something CHANGED. Queue time becoming a named flow-control
    // hold changes the substance of the bar without changing the state of the
    // row, so the second mark would repeat the first and is dropped.
    //
    // Unless the SUBSTANCE is what changed hands. A checkpointed step opens
    // with the request's own work and then begins its own -- two `started`
    // moments, one carrying `disc` and one not -- and that is the boundary
    // between Inngest executing and your step executing, which is the single
    // most important thing a row says. Dropping it left the handover to be
    // inferred from a colour change with no mark on it.
    const handover = out.length && out[out.length-1].s==='disc' && !sub;
    if(out.length && out[out.length-1].c===c && !handover) continue;
    out.push({p:x, c, s:sub});
  }
  return out;
}

/**
 * What a slice of the Run row is worth, and which wins where slices overlap.
 *
 * The row is a profile of where the elapsed time went: grey wherever nothing
 * was executing, and coloured wherever something was. Which colour is a
 * priority, not an average -- several things can be running at one instant and
 * an overview has to answer "what is the worst thing happening here":
 *
 *   3  a failure, red -- the thing an overview exists to make findable
 *   2  returned, green
 *   1  your compute in use, blue -- a step still running or the SDK reporting.
 *      It ranks last so it never hides an outcome, but it IS a rank, so a
 *      moment where the only thing happening is a discovery request reads blue
 *      instead of dropping out of the profile altogether.
 *   0  ended without an outcome, grey
 *
 * Neutral "mix" is deliberately absent. Averaging a failure with the successes
 * around it rendered a failure cluster blue, losing the signal at exactly the
 * scale the row exists for.
 */
export const RUN_RANK = {
  bad:3, spanerr:3,
  good:2, child:2, spanok:2,
  running:1, disc:1,
  stopped:0, spanunset:0,
};

/** The rank of a bar, or null if it is not compute and does not colour the row. */
export function runRank(kind){
  const k=kind.replace(/[!*]+$/,'');
  if(/!\*?$/.test(kind) || k==='bad') return RUN_RANK.bad;
  const v=RUN_RANK[k];
  return v===undefined?null:v;
}

/** Rank to colour. */
export const RUN_COLOUR = ['var(--muted)','var(--disc)','var(--good)','var(--warn)'];

/**
 * What stays lit when a row is hovered or selected.
 *
 * Attention is per ELEMENT, not per row: hovering a step lights the parts of
 * other rows that caused it and leaves the rest of those rows faded. Fading a
 * whole row would either hide the cause or light work that had nothing to do
 * with it.
 *
 * Three things survive, and they are the three answers to "why is this row
 * here":
 *
 *   - the row itself, entire;
 *   - on each row a cable comes from, the interval that ENDS where the cable
 *     starts -- the work whose finishing queued the request -- and the two
 *     moments bounding it. Not that row's other work, which is unrelated;
 *   - if the row was planned by a request drawn on another row, that request:
 *     its bars and moments up to the point it resolved, and the planned mark
 *     the ribbon threads.
 *
 * Returns a map of row index to `true` (all of it) or {bars, dots} of the
 * positions that survive. Everything absent is dimmed.
 */
export function attention(rows, focus){
  const out=new Map();
  if(focus==null || focus<0 || !rows[focus]) return out;
  out.set(focus, true);

  for(const c of cables(rows, focus)){
    const src=rows[c.from];
    const at=src.at||[];
    // the moment the interval ending at the cable's start began
    let i=at.length-1;
    while(i>0 && at[i][1]>c.x-1e-9) i--;
    const a=at[i], b=at.find(m=>Math.abs(m[1]-c.x)<1e-9)||at[i+1];
    if(!a||!b) continue;
    out.set(c.from, {bars:[+a[1].toFixed(3)], dots:[+a[1].toFixed(3), +b[1].toFixed(3)]});
  }

  const g=ribbonGroups(rows).find(x=>x.rows.includes(focus));
  if(g){
    const owner=g.rows.find(i=>i!==focus && rows[i] && rows[i].reported);
    if(owner!=null && !out.has(owner)){
      const at=rows[owner].at||[];
      const k=at.findIndex(m=>RESOLVED.has(m[0]));
      const head=k<0?at:at.slice(0,k+1);
      out.set(owner, {
        bars:head.slice(0,-1).map(m=>+m[1].toFixed(3)),
        dots:[...new Set(head.map(m=>+m[1].toFixed(3)).concat([g.x]))],
      });
    }
  }
  return out;
}
