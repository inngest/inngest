/**
 * THE SINGLE SOURCE OF TRUTH for the trace design language.
 *
 * Both renderers import from here. Change a colour, a mark, or a rule in this
 * file and every figure in the artifact changes with it — there is no second
 * copy to keep in sync, which is how the two renderers drifted before.
 *
 * Two axes, twice over:
 *   BARS    colour = what kind of work; solid or hatched = did your app execute
 *   CIRCLES colour = what kind of moment; hollow or filled = has it finished
 */

/**
 * The geometry of a row, in the one place that defines it. Every renderer,
 * including the one that draws the scrubbable fixtures in the browser, reads
 * these rather than restating them.
 */
export { GEOM } from './rules.mjs';
import { GEOM } from './rules.mjs';
export const cyOf = i => GEOM.TOP + 9 + GEOM.ROW * i;
export const pxOf = (p, k=1) => GEOM.LBL + ((p/100)*GEOM.PLOT)*k;

export const C = {
  good:'var(--good)', bad:'var(--warn)', disc:'var(--disc)', idle:'var(--rule-2)',
  child:'var(--child)', hold:'var(--hold)',
  acc:'var(--accent)', mut:'var(--muted)', mix:'var(--disc)', queued:'var(--queued)', ink2:'var(--ink-2)',
};

/**
 * The semantic colour each bar kind takes. Every kind is then indirected
 * through its own `--bar-<kind>` variable so the configurator can move one
 * kind without moving everything that shares its meaning.
 */
const SEMANTIC = {
  good:C.good, bad:C.bad, disc:C.disc, idle:C.idle,
  child:C.child, mix:C.mix, running:C.disc,
  wait:C.disc, waitok:C.good, waitout:C.mut, backoff:C.bad,
  spanok:C.good, spanerr:C.bad, spanunset:C.mut,
  stopped:C.mut, waitstop:C.mut, hold:C.hold,
};
export const FILL = Object.fromEntries(
  Object.entries(SEMANTIC).map(([k,v])=>[k,`var(--bar-${k}, ${v})`]));

/** Your app executed for these. Everything else is drawn hatched. */
export const COMPUTE = new Set(['good','bad','child','running','disc','stopped','spanok','spanerr','spanunset']);
export const NOCOMPUTE = new Set(['wait','waitok','waitout','idle','backoff','waitstop','hold']);
/** Kinds that represent work happening, for the resolution rule. */
export const ACTIVE = new Set(['good','bad','child','running','disc','stopped','spanok','spanerr','spanunset']);

/** A kind may be suffixed `!` to mean "this substance, but it failed". */
export const base = k => k.replace(/[!*]+$/,'');
export const discovered = k => k.includes('*');
export const isFail = k => /!\*?$/.test(k) || base(k)==='bad';

/**
 * Hatch weights per substance. Backoff is red, drawn faded: it is a consequence
 * of a failure rather than a failure itself, so it carries the cause without
 * carrying the alarm, and at full strength a recovered run looked more broken
 * than a run that actually failed.
 *
 * NOTE: this contradicts LOG.md critic round 4, which settled backoff as
 * NEUTRAL, and the React implementation, where `timing.backoff` is
 * `bg-surfaceMuted` and deliberately untinted. The artifact is the stakeholder's
 * call and wins here; one of the other two records is stale and needs deciding.
 */
const HATCH_WEIGHT = {
  // spanok/spanerr/spanunset are solid: a userland span is your code running.
  wait:   {bg:.16, line:.55},
  waitok: {bg:.16, line:.55},
  waitout:{bg:.16, line:.55},
  hold:   {bg:.16, line:.62},
  idle:   {bg:.30, line:1},
  backoff:{bg:.12, line:.42},
  waitstop:{bg:.16, line:.55},
};

/**
 * Every kind is painted with a pattern, solid ones included: a solid bar is
 * the same tile with its background at full strength and no stripe. That makes
 * "did your app execute" one pair of numbers per kind rather than two
 * different painting mechanisms, which is what lets the configurator move a
 * kind between them without regenerating anything.
 */
export const SUBSTANCE = Object.fromEntries(Object.keys(SEMANTIC).map(k=>
  [k, NOCOMPUTE.has(k) ? (HATCH_WEIGHT[k] || {bg:.16, line:.55}) : {bg:1, line:0}]));

export const OPEN = new Set(['running','wait']);

export const HATCH = `<defs>`+
  `<filter id="hx-wire" x="-20%" y="-40%" width="140%" height="220%">`+
  `<feDropShadow dx="0" dy="1.4" stdDeviation="1.1" flood-color="#000" flood-opacity=".6"/></filter>`+
  `<linearGradient id="hx-open" x1="0" x2="1">`+
  `<stop offset="0" stop-color="var(--surface)" stop-opacity="0"/>`+
  `<stop offset="1" stop-color="var(--surface)" stop-opacity="1"/></linearGradient>`+
  Object.keys(SUBSTANCE).map(k=>
    `<pattern id="hx-${k}" width="4" height="4" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">`+
    `<rect class="sb sb-${k}" width="4" height="4" fill="${FILL[k]}"/>`+
    `<line class="sl sl-${k}" x1="1" y1="0" x2="1" y2="4" stroke="${FILL[k]}"/></pattern>`).join('')+
  `</defs>`;

/** What a bar of this kind is painted with. */
export const paint = k => `url(#hx-${base(k)})`;

/** The `:root` defaults the configurator overrides and resets back to. */
export const substanceCSS = () =>
  Object.entries(SUBSTANCE).map(([k,{bg,line}])=>
    `.sb-${k}{opacity:var(--sub-${k}-bg,${bg})}`+
    `.sl-${k}{opacity:var(--sub-${k}-line,${line});stroke-width:var(--sub-${k}-w,2)}`).join('\n  ');

/**
 * The event marks. Hollow means the step has not finished; filled means it has.
 * Only a resolution is ever green or red.
 */
export const EV = {
  queued:'queued', started:'hollow', retry:'hollow-bad', ribbon:'ribbon', done:'done',
  ok:'ok', failed:'failed', timeout:'timeout', cancelled:'cancelled',
};

/** Each mark's own variable, defaulting to the semantic colour it means. */
const EV_SEMANTIC = {
  queued:C.queued, ribbon:C.disc, hollow:C.mut, 'hollow-bad':C.bad, done:C.mut,
  ok:C.good, failed:C.bad, timeout:C.mut, cancelled:C.mut,
};
export const EVC = Object.fromEntries(
  Object.entries(EV_SEMANTIC).map(([k,v])=>[k,`var(--ev-${k}, ${v})`]));
/** Filled marks mean finished; hollow ones mean the row has not resolved. */
export const EV_HOLLOW = ['queued','ribbon','hollow','hollow-bad'];

export function markSvg(x, y, c, o=1, r=GEOM.MARK_R, halo=true, cp=false, sub=''){
  const R=r+GEOM.HALO;
  const col=EVC[c]||c;
  /**
   * The checkpoint ring, outside everything else the mark draws.
   *
   * Outside the halo too, so the halo still does its job of putting a ring of
   * surface between the mark and whatever it sits on. A square mark
   * (`cancelled`) gets a square ring, so the silhouette stays the one thing
   * that says "stopped from outside".
   */
  const ring = !cp ? ''
    : c==='cancelled'
      ? `<rect class="ev-cp sq" x="${x-r-GEOM.CP_GAP}" y="${y-r-GEOM.CP_GAP}" width="${(r+GEOM.CP_GAP)*2}" height="${(r+GEOM.CP_GAP)*2}" rx="1" fill="none" stroke="${col}" stroke-width="${GEOM.CP_W}" opacity="${o*GEOM.CP_O}"/>`
      : `<circle class="ev-cp" cx="${x}" cy="${y}" r="${r+GEOM.CP_GAP}" fill="none" stroke="${col}" stroke-width="${GEOM.CP_W}" opacity="${o*GEOM.CP_O}"/>`;
  const bg = !halo ? ''
    : c==='cancelled'
      ? `<rect class="ev-bg sq" x="${x-R}" y="${y-R}" width="${R*2}" height="${R*2}" fill="var(--surface)" opacity="${o}"/>`
      : `<circle class="ev-bg" cx="${x}" cy="${y}" r="${R}" fill="var(--surface)" opacity="${o}"/>`;
  if(c==='cancelled')
    return bg+ring+`<rect class="ev ev-cancelled"${sub?` data-sub="${sub}"`:''} x="${x-r}" y="${y-r}" width="${r*2}" height="${r*2}" rx="0.8" fill="${col}" stroke="${col}" opacity="${o}"/>`;
  if(EV_HOLLOW.indexOf(c)>=0)
    return bg+ring+`<circle class="ev ev-${c}"${sub?` data-sub="${sub}"`:''} cx="${x}" cy="${y}" r="${r}" fill="var(--surface)" stroke="${col}" opacity="${o}"/>`;
  return bg+ring+`<circle class="ev ev-${c}"${sub?` data-sub="${sub}"`:''} cx="${x}" cy="${y}" r="${r}" fill="${col}" stroke="${col}" opacity="${o}"/>`;
}
export const dot = markSvg;


/**
 * Ring weights are per mark — `queued` and `retry` were drawn heavier than the
 * rest on purpose — and each is overridable.
 */
const EV_WEIGHT = {queued:1.8, ribbon:1.7, disc:1.8, hollow:1.7, 'hollow-bad':1.9};
export const eventCSS = () =>
  Object.keys(EV_SEMANTIC).map(k=>
    `.ev-${k}{stroke-width:var(--ev-${k}-w,${EV_WEIGHT[k]||1})}`).join('\n  ');

/**
 * One bar. Every kind is a pattern, so this is the only place a bar is drawn.
 * `k` is the axis scale, so the minimum width is a minimum on screen.
 */
export function barSvg(kind, x, w, y, {k=1, floor=k, o=1, h=GEOM.BAR_H}={}){
  // `k` scales coordinates now; `floor` is the scale the drawing ends up at,
  // so the minimum width is a minimum on screen whether the caller scales
  // afterwards (the static figures) or not (the browser).
  const b=kind.replace(/[!*]+$/,'');
  // y is the row's centre line; the CSS centres the bar on it, so changing the
  // height in the panel does not also move it.
  const cls=h===GEOM.BAR_H?'bar':(h<3?'bar rail':'bar sm');
  const wpx=Math.max(GEOM.MIN_W*k/floor,(w/100)*GEOM.PLOT*k);
  return `<rect class="${cls}" style="--w:${wpx.toFixed(2)}px" `+
    `x="${pxOf(x,k).toFixed(2)}" y="${y.toFixed(1)}" `+
    `width="${wpx.toFixed(2)}" height="${h}" `+
    `rx="1" fill="url(#hx-${b})" opacity="${o}"/>`;
}

/**
 * A circle at every boundary, coloured by what starts there. A failed interval
 * ends on a retry mark; the last boundary is the resolution — unless the row is
 * still running, in which case there is nothing to resolve and no mark.
 *
 * `segs` is [{kind,x,w}] in plot percent.
 */
/**
 * Fill any gap between segments with queued time. A gap is unexplained time,
 * which the design does not allow — and left as a bare gap it also swallowed
 * the mark for the step starting.
 */
export function fillGaps(segs){
  const out=[];
  segs.forEach((g,i)=>{
    const prev=segs[i-1];
    if(prev){
      const end=prev.x+prev.w;
      if(g.x-end>0.01) out.push({kind:'idle',x:end,w:g.x-end});
    }
    out.push(g);
  });
  // Two abutting segments of the same kind are one interval, not two, and
  // would otherwise draw a boundary mark where nothing changed.
  const merged=[];
  for(const g of out){
    const last=merged[merged.length-1];
    if(last && last.kind===g.kind && Math.abs(last.x+last.w-g.x)<0.01) last.w+=g.w;
    else merged.push({...g});
  }
  return merged;
}

export function autoDots(segs){
  if(!segs.length) return [];
  const out=[];
  const kindDot = k =>
    base(k)==='disc' ? EV.started :
    (base(k)==='idle'||base(k)==='backoff'||base(k)==='hold') ? EV.queued :
    EV.started;
  segs.forEach((g,j)=>{
    const prev=segs[j-1];
    const gap = prev && Math.abs(prev.x+prev.w-g.x) > 0.01;
    if(gap) out.push({p:prev.x+prev.w, c:isFail(prev.kind)?EV.retry:EV.queued});
    out.push({p:g.x, c:
      j===0 ? EV.queued :
      (prev && isFail(prev.kind) && !gap) ? EV.retry :
      kindDot(g.kind)});
  });
  const last=segs[segs.length-1], lastK=base(last.kind);
  if(!OPEN.has(lastK))
    out.push({p:last.x+last.w, c:
      (lastK==='stopped'||lastK==='waitstop') ? EV.cancelled :
      lastK==='spanerr' ? EV.failed :
      lastK==='spanunset' ? EV.done :
      isFail(last.kind) ? EV.failed :
      lastK==='waitout' ? EV.timeout :
      (ACTIVE.has(lastK)||lastK==='waitok') ? EV.ok : EV.queued});
  // A mark says something changed. Where a bar changes substance without
  // changing state — queued time becoming a named flow-control hold — the
  // second mark would repeat the first, so it is dropped.
  return out.filter((d,i)=> i===0 || d.c!==out[i-1].c);
}


/**
 * The outbound lineage stub.
 *
 * A row whose work started runs elsewhere ends in a short spur to a hollow
 * ring and a count. The ring is hollow because those runs are not in this
 * trace: it is a door, not a span. Nothing else in the system leaves the row
 * to the right, so the shape is unambiguous.
 */
export function lineage(x, y, count, {o=1}={}){
  const label = count + ' run' + (count===1?'':'s') + ' \u2197';
  return `<g opacity="${o}">`+
    `<path d="M${x+3} ${y} l7 0" stroke="${C.acc}" stroke-width="1.5" fill="none"/>`+
    `<circle cx="${x+13}" cy="${y}" r="3" fill="none" stroke="${C.acc}" stroke-width="1.5"/>`+
    `<circle cx="${x+13}" cy="${y}" r="1" fill="${C.acc}"/>`+
    `<text x="${x+19}" y="${y+3}" font-family="JetBrains Mono, ui-monospace, monospace" font-size="7" fill="${C.acc}">${label}</text></g>`;
}

/**
 * A causal run, drawn in the same orthogonal language as the grouping ribbon:
 * out along each contributor's own row, down one shared spine, in to where the
 * step actually starts.
 *
 * The spine sits at the LAST contributor's end, because that is the moment the
 * next discovery request was queued — the same inference the grouping ribbon
 * makes when it snaps members to a single queue instant. No invented offset.
 *
 * Hatched, so it never reads as the solid grouping ribbon: that one is
 * always-there structure, this one is a temporary answer to a question.
 */
export function causalRibbon(sources, target, { px, cy, o = 1, w = 2.4 } = {}) {
  if (!sources.length) return '';
  const ys = sources.map(s => cy(s.row));
  const xJ = px(Math.max(...sources.map(s => s.x)));
  const yT = cy(target.row), xT = px(target.x);
  const line = d => `<path d="${d}" fill="none" stroke="${C.disc}" stroke-width="${w}" stroke-dasharray="4 3" stroke-linecap="round" opacity="${o}"/>`;
  let out = '';
  sources.forEach((s, i) => { if (px(s.x) < xJ - 0.5) out += line(`M${px(s.x)} ${ys[i]} H${xJ}`); });
  out += line(`M${xJ} ${Math.min(...ys, yT)} V${Math.max(...ys, yT)}`);
  if (xT - 4 > xJ + 0.5) out += line(`M${xJ} ${yT} H${xT - 4}`);
  out += `<path d="M${xT - 4} ${yT - 3} L${xT} ${yT} L${xT - 4} ${yT + 3} Z" fill="${C.disc}" opacity="${o}"/>`;
  return out;
}


/**
 * A wire. Thin, pale, curved, and lifted off the plane by a soft shadow so it
 * reads as a cable running between rows rather than a line drawn on them. It
 * ends at the edge of the hollow circle it connects to, which makes that
 * circle a socket the wire plugs into — hence no arrowhead: a plug does not
 * need to point.
 */
export function wire(x1, y1, x2, y2, { o = 1, r = 3, i = 0 } = {}) {
  const span = Math.abs(x2 - x1);
  // Vary the bow slightly per wire so several plugging into one socket separate
  // along their length instead of running parallel into it.
  const dx = Math.max(5, Math.min(span * 0.55, 34)) * (1 + i * 0.34);
  const x = x1 + r + 1.2, end = x2 - r - 1.2;
  const d = `M${x} ${y1} C ${x + dx} ${y1}, ${end - dx} ${y2}, ${end} ${y2}`;
  // A dark casing under the pale core, so a crossing reads as one cable
  // passing over another rather than two lines merging.
  const P='vector-effect="non-scaling-stroke"';
  const a=((y1-(GEOM.TOP+9))/GEOM.ROW).toFixed(4);
  return `<g class="cable" style="--a:${a};transform-origin:0 ${y1.toFixed(1)}px" `+
    `filter="url(#hx-wire)" opacity="${o}">`+
    `<path d="${d}" fill="none" stroke="var(--ground)" stroke-width="3.2" stroke-linecap="round" opacity=".85" ${P}/>`+
    `<path d="${d}" fill="none" stroke="${C.ink2}" stroke-width="1.5" stroke-linecap="round" ${P}/></g>`;
}


/**
 * Stretch a finished figure so its content fills the plot.
 *
 * Specs are hand-authored in percentages that rarely reach 100, which left a
 * quarter of every canvas empty. Rather than renumber every fragment — and
 * every ribbon and cable computed from the same numbers — this scales the
 * finished SVG about the left edge of the plot, so bars, circles, ribbons and
 * cables all move together. Radii and stroke widths are left alone.
 */
/**
 * Rewrite every x in a generated fragment through a mapping of the time axis.
 *
 * Rows can be laid out before they are drawn, but arrows, ribbons, cables and
 * annotations are handed to `fig` already drawn, in pixels. Without this they
 * stay on the old axis and point at nothing — which is why compression could
 * only ever be applied to figures that had none of them.
 *
 * Modelled on `stretch`, which already proves the technique: the same set of
 * attributes, the same path walk. Anything left of the label gutter is a label
 * and does not move.
 */
export function remapX(svg, LBL, at){
  const X = v => {
    const n = parseFloat(v);
    if(!(n > LBL)) return String(n);
    const p = (n - LBL) * 100 / GEOM.PLOT;
    return (LBL + at(p) / 100 * GEOM.PLOT).toFixed(2);
  };
  const defs = svg.match(/<defs>[\s\S]*?<\/defs>/);
  const head = defs ? defs[0] : '';
  let body = defs ? svg.replace(head, '\u0000DEFS\u0000') : svg;
  body = body
    .replace(/<rect ([^>]*?)x="([\d.-]+)"([^>]*?)width="([\d.]+)"/g,
      (m,a,x,b,w) => {
        const x0=parseFloat(x), x1=x0+parseFloat(w);
        const n0=parseFloat(X(x0)), n1=parseFloat(X(x1));
        return `<rect ${a}x="${n0.toFixed(2)}"${b}width="${Math.max(0.4,n1-n0).toFixed(2)}"`;
      })
    .replace(/--w:([\d.]+)px/g, m => m)   // recomputed below by the width rewrite
    .replace(/cx="([\d.-]+)"/g, (m,x) => `cx="${X(x)}"`)
    .replace(/<text ([^>]*?)x="([\d.-]+)"/g, (m,a,x) => `<text ${a}x="${X(x)}"`)
    .replace(/x1="([\d.-]+)"/g, (m,x) => `x1="${X(x)}"`)
    .replace(/x2="([\d.-]+)"/g, (m,x) => `x2="${X(x)}"`)
    .replace(/ d="([^"]+)"/g, (m,d) => {
      let out='', cmd='';
      const toks = d.match(/[A-Za-z]|-?[\d.]+/g) || [];
      const nums = [];
      const flush = () => {
        if(!cmd) return;
        if(cmd==='H') out += ' H' + nums.map(X).join(' ');
        else if(cmd==='V') out += ' V' + nums.join(' ');
        else if(cmd==='Z') out += ' Z';
        else out += ' ' + cmd + nums.map((v,j)=> j%2===0 ? X(v) : v).join(' ');
        nums.length = 0;
      };
      for(const t of toks){
        if(/[A-Za-z]/.test(t)){ flush(); cmd = t; }
        else nums.push(t);
      }
      flush();
      return ` d="${out.trim()}"`;
    });
  return defs ? body.replace('\u0000DEFS\u0000', head) : body;
}

export function stretch(svg, LBL, k){
  if(!(k>1.001)) return svg;
  const X = v => { const n = parseFloat(v); return n < LBL ? String(n) : (LBL + (n - LBL) * k).toFixed(2); };
  const defs = svg.match(/<defs>[\s\S]*?<\/defs>/);
  const head = defs ? defs[0] : '';
  let body = defs ? svg.replace(head, '\u0000DEFS\u0000') : svg;

  body = body
    .replace(/--w:([\d.]+)px/g, (m,w) => `--w:${(parseFloat(w)*k).toFixed(2)}px`)
    .replace(/<rect ([^>]*?)x="([\d.-]+)"([^>]*?)width="([\d.]+)"/g,
      (m,a,x,b,w) => `<rect ${a}x="${X(x)}"${b}width="${(parseFloat(w)*k).toFixed(2)}"`)
    .replace(/cx="([\d.-]+)"/g, (m,x) => `cx="${X(x)}"`)
    .replace(/<text ([^>]*?)x="([\d.-]+)"/g, (m,a,x) => `<text ${a}x="${X(x)}"`)
    .replace(/x1="([\d.-]+)"/g, (m,x) => `x1="${X(x)}"`)
    .replace(/x2="([\d.-]+)"/g, (m,x) => `x2="${X(x)}"`)
    // path data: every command here is absolute, so scale the x of each pair
    .replace(/ d="([^"]+)"/g, (m,d) => {
      let out='', i=0, cmd='';
      const toks = d.match(/[A-Za-z]|-?[\d.]+/g) || [];
      const nums = [];
      const flush = () => {
        if(!cmd) return;
        if(cmd==='H') out += ' H' + nums.map(X).join(' ');
        else if(cmd==='V') out += ' V' + nums.join(' ');
        else if(cmd==='Z') out += ' Z';
        else out += ' ' + cmd + nums.map((v,j)=> j%2===0 ? X(v) : v).join(' ');
        nums.length = 0;
      };
      for(const t of toks){
        if(/[A-Za-z]/.test(t)){ flush(); cmd = t; }
        else nums.push(t);
      }
      flush();
      return ` d="${out.trim()}"`;
    });

  return defs ? body.replace('\u0000DEFS\u0000', head) : body;
}


/** What each element is called and what it means, for the row breakdown. */
/**
 * `running` and `disc` are ONE substance with two names.
 *
 * They were listed separately and drawn identically, which asked a reader to
 * learn a distinction the drawing does not make. Blue means the same thing in
 * both cases and it is the useful thing: your compute is in use right now --
 * a step executing, or the SDK being asked what runs next. Which of the two it
 * is, is already said by the row it sits on.
 */
export const BAR_INFO={
  running:['running / discovery','your app is executing — a step running, or the SDK reporting what runs next'],
  good:['ok','step.run() returned'],
  bad:['failed','step.run() threw'],
  stopped:['stopped','executing when the run was cancelled'],
  child:['child','step.invoke() child run'],
  disc:['running / discovery','your app is executing — a step running, or the SDK reporting what runs next'],
  wait:['waiting','step.sleep(), step.waitForEvent() or step.waitForSignal(), not resolved'],
  waitok:['sleep / matched','step.sleep() elapsed, or a wait matched'],
  waitout:['timed out','the wait expired with no match'],
  waitstop:['stopped','waiting when the run was cancelled'],
  backoff:['backoff','retry backoff before the next attempt'],
  spanok:['span ok','a userland span that reported OK'],
  spanerr:['span error','a userland span that reported ERROR'],
  spanunset:['span','a userland span that set no status — the common case'],
  hold:['held','held by concurrency, throttle, rate limit or debounce'],
  idle:['queued','waiting for the executor to pick it up'],
};

export const EVENT_INFO={
  queued:['queued','enqueued, not started'],
  ribbon:['planned','enqueued by a discovery request that reported several steps'],
  hollow:['started','execution started'],
  'hollow-bad':['retry','the attempt threw, another will follow'],
  done:['done','resolved, and no status was reported — OpenTelemetry Unset'],
  ok:['ok','resolved, succeeded'],
  failed:['failed','threw on the final attempt'],
  timeout:['timeout','the wait expired with no match'],
  cancelled:['cancelled','the run was cancelled'],
};
