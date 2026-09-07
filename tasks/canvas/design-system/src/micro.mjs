import { GEOM } from './vocabulary.mjs';
export const {W,LBL,RGT,PLOT,ROW,TOP}=GEOM;
export { GEOM };
export { C, EV, HATCH, dot, base, isFail, FILL, COMPUTE, NOCOMPUTE, ACTIVE, OPEN, paint, autoDots, discovered, lineage, barSvg, markSvg, cyOf, pxOf, BAR_INFO, EVENT_INFO, EVC, SUBSTANCE, substanceCSS, eventCSS } from './vocabulary.mjs';
import { C, EV, HATCH, dot, base, paint, autoDots, OPEN, COMPUTE, isFail, barSvg, causalRibbon as _cr, wire, fillGaps, stretch, BAR_INFO, EVENT_INFO } from './vocabulary.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";
let NOTES=false;
export const setNotes=on=>{NOTES=on;};
export const px=p=>LBL+(p/100)*PLOT;
export const cy=i=>TOP+9+ROW*i;
/** Row pitch for a userland span: tight enough that the bars nearly touch. */
export const SPAN_ROW=9;
/** y for each row, honouring any that sit on the tighter span pitch. */
export const rowYs=rows=>{
  const ys=[]; let y=cy(0);
  rows.forEach((r,i)=>{ if(i) y+=rows[i].span?SPAN_ROW:(rows[i-1].span?SPAN_ROW+4:ROW); ys.push(y); });
  return ys;
};
export const arrow=(x1,y1,x2,y2,o=1)=>{
  const d=Math.max(10,Math.abs(x2-x1)*0.5);
  return `<path d="M${x1} ${y1} C ${x1+d} ${y1}, ${x2-d} ${y2}, ${x2-3} ${y2}" fill="none" stroke="${C.acc}" stroke-width="1.3" opacity="${o}"/>`+
    `<path d="M${x2-3} ${y2-2.2} L${x2} ${y2} L${x2-3} ${y2+2.2} Z" fill="${C.acc}" opacity="${o}"/>`;
};
export { wire };
export const causal=(sources,target,opts={})=>_cr(sources,target,{px,cy,...opts});
export const tag=(x,y,t,c=C.mut)=>`<text x="${x}" y="${y}" ${MONO} font-size="6.5" fill="${c}">${t}</text>`;

/**
 * A row. `segs` are [kind, start%, width%]. Circles are placed automatically at
 * transitions: the row's first event, each point where active work begins after
 * idle time or changes outcome, and the resolution — which is the only one that
 * carries status.
 */
/**
 * The Run row: a profile of where the elapsed time went. It replaced the
 * separate minimap above it, which drew the same run in the same place in the
 * same colours — two drawings of one fact, which is the
 * characteristic defect of this view, and a second overview that could drift
 * out of step with the first.
 *
 * **Failure wins over mixed.** A slice holding both a success and a failure
 * draws red, not the neutral `mix`. The old "only if every step agrees" rule
 * made a failure cluster among successes render blue — losing the signal at
 * exactly the scale an overview exists for. The row already bent that way:
 * failed slices have always had a wider minimum so they stay findable.
 */
export function runProfile(i,{to=86,intervals=[],resolved,n='Run',breaks=[]},sc=1){
  const y=cy(i);
  const col=v=> v.ok===false?C.bad : v.ok==='stop'?C.mut : v.ok==='mix'?C.mix : C.good;
  let s=`<text x="2" y="${y+2.5}" ${MONO} font-size="7" fill="${C.mut}">${n}</text>`;
  /**
   * The grey track is the run's whole extent — and where the axis is
   * compressed, **the track itself tears** rather than running straight under a
   * separate squiggle laid on top. A glyph beside the bar reads as an icon; the
   * bar breaking reads as the thing that happened to it.
   */
  const end=LBL+(to/100)*PLOT;
  const tear=(x0,x1)=>{
    // Small. The tear only has to be legible as a break in the track, and at
    // any real amplitude it stops being the track and becomes a decoration
    // sitting where the track used to be.
    const n=Math.min(3,Math.max(2,Math.round((x1-x0)/8))), step=(x1-x0)/n, amp=1.9;
    let d=`M${x0.toFixed(1)} ${y}`;
    for(let j=0;j<n;j++)
      d+=` L${(x0+step*(j+0.5)).toFixed(1)} ${(y+(j%2?amp:-amp)).toFixed(1)}`+
         ` L${(x0+step*(j+1)).toFixed(1)} ${y}`;
    return `<path d="${d}" fill="none" stroke="${C.idle}" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>`;
  };
  {
    let at=LBL;
    for(const [a,b] of breaks){
      const x0=Math.max(LBL,px(a)), x1=Math.min(end,px(b));
      if(x1<=x0) continue;
      if(x0>at) s+=`<rect x="${at.toFixed(1)}" y="${y-2.5}" width="${(x0-at).toFixed(1)}" height="5" rx="1.2" fill="${C.idle}"/>`;
      s+=tear(x0,x1); at=x1;
    }
    if(end>at) s+=`<rect x="${at.toFixed(1)}" y="${y-2.5}" width="${(end-at).toFixed(1)}" height="5" rx="1.2" fill="${C.idle}"/>`;
  }
  // Failures last, so where slices overlap the red is the one left showing.
  const ordered=[...intervals].sort((p,q)=>(p.ok===false?1:0)-(q.ok===false?1:0));
  for(const v of ordered){
    const a=v.a, b=Math.min(v.b,to);
    if(b<=a) continue;
    // Failure stays findable even when the interval is tiny.
    const running=v.b>to;
    const w=Math.max((v.ok===false&&!running?3.2:1.4)/sc, ((b-a)/100)*PLOT);
    s+=`<rect x="${px(a).toFixed(1)}" y="${y-4}" width="${w.toFixed(1)}" height="8" rx="1.2" fill="${running?C.disc:col(v)}"/>`;
  }
  s+=dot(LBL,y,EV.queued);
  if(resolved) s+=dot(LBL+(to/100)*PLOT,y,resolved);
  return s;
}

export function row(i,r,sc=1,yy){
  if(r.run) return runProfile(i,r,sc);
  const {n,segs:rawSegs=[],dim=1,dots,note,sel,noHalo,lit,litDots}=r;
  const segs=fillGaps(rawSegs.map(([k,x,w])=>({kind:k,x,w}))).map(g=>[g.kind,g.x,g.w]);
  const y=yy!=null?yy:cy(i); let s='', hit='';
  /**
   * Attention is per ELEMENT, not per row. Hovering a step lights the parts of
   * other rows that caused it — the discovery bar, the queue circle, the mark
   * the ribbon threads — and leaves the rest of those rows faded. Fading a
   * whole row would either hide the cause or light an `ok` bar that had nothing
   * to do with it.
   *
   * `lit` names what survives: a bar kind (`'disc'`) or a segment's start x.
   * `litDots` names mark positions. Absent, the row is uniformly `dim`.
   */
  const litBar=(k,x)=>!lit ? dim
    : (lit.some(v=>typeof v==='string' ? v===base(k) : Math.abs(v-x)<0.01) ? 1 : dim);
  const litDot=p=>!litDots ? dim : (litDots.some(v=>Math.abs(v-p)<0.01) ? 1 : dim);
  if(sel) s+=`<rect x="${LBL-3}" y="${y-7.5}" width="${PLOT+6}" height="15" fill="${C.acc}" opacity=".13" rx="2"/>`;
  /**
   * Fading happens on a GROUP, not on each element. Per-element opacity made a
   * dimmed row translucent to ITSELF: a mark's halo is a disc of surface
   * colour, so at .15 the bar behind showed straight through it and the row
   * lost its layering exactly where it was carrying the most meaning.
   * Compositing the row and fading the result keeps every mark on top of its
   * bar, at any opacity.
   *
   * Two groups rather than one, because attention is per element: the parts of
   * a row that caused the hovered one stay at full strength while the rest of
   * that same row fades. The dim group is emitted first, so a lit mark still
   * lands over a dim bar.
   *
   * A mark on a ribbon keeps its halo. It was suppressed so the circle would
   * read as sitting ON the ribbon, but that cost it the ring of surface every
   * other mark has. The ribbon threads BETWEEN the halos instead.
   */
  const auto=autoDots(segs.map(([k,a,w])=>({kind:k,x:a,w})));
  let lo='', hi='';
  // The dim layer is the WHOLE row, not the leftover after the lit parts are
  // taken out. Splitting it that way separated bars from marks into different
  // groups, so a lit bar was drawn OVER a dim mark that belongs on top of it.
  // Drawing the complete row at `dim` and then repainting only the lit parts
  // over it keeps bars under marks inside both layers.
  const put=(on,frag)=>{ lo+=frag; if(on) hi+=frag; };
  if(r.spans) lo+=[0,1,2].map(j=>
    `<rect x="${LBL-11}" y="${(y-2.6+j*2.2).toFixed(1)}" width="6" height="1.1" rx="0.5" fill="${C.mut}" opacity="${0.85-j*0.22}"/>`).join('');
  lo+=`<text x="${r.span?8:2}" y="${y+2.2}" ${MONO} font-size="${r.span?6:7}" `+
     `fill="${C.mut}" opacity="${r.span?0.78:1}">${n}</text>`;
  // A userland span is a subdivision of the step above it, not a peer, so it is
  // drawn thinner. Nesting and weight carry that, not a new colour: an OTel span
  // IS your code, so it keeps the same status colours the step has.
  const bh=(r.thin||r.span)?GEOM.BAR_H*0.6:GEOM.BAR_H;
  segs.forEach(([k,a,w])=>{ put(litBar(k,a)===1, barSvg(k,a,w,y,{k:1,floor:sc,h:bh})); });
  if(!r.span) (dots||auto).forEach(d=>{
    const onRib=(noHalo||[]).some(p=>Math.abs(p-d.p)<0.01);
    put(litDot(d.p)===1, dot(px(d.p),y,onRib?'ribbon':(d.c||C.mut),1,3,true));
  });
  // The note follows the row's own content rather than sitting in a reserved
  // column, so no horizontal space is set aside for it.
  {
    const parts=[];
    const marks=(dots||auto);
    let mi=0;
    segs.forEach(g=>{
      while(mi<marks.length && marks[mi].p<=g[1]+0.001){
        const e=EVENT_INFO[marks[mi].c]; if(e) parts.push({t:'e',k:marks[mi].c,n:e[0],d:e[1]});
        mi++;
      }
      const b=BAR_INFO[base(g[0])]; if(b) parts.push({t:'b',k:base(g[0]),n:b[0],d:b[1]});
    });
    for(;mi<marks.length;mi++){ const e=EVENT_INFO[marks[mi].c]; if(e) parts.push({t:'e',k:marks[mi].c,n:e[0],d:e[1]}); }
    if(parts.length) hit=`<rect class="rowhit" x="${LBL-6}" y="${y-8}" width="${PLOT+12}" height="16" fill="transparent" data-row="${n}" data-parts='${JSON.stringify(parts).replace(/'/g,"&apos;")}'/>`;
  }
  if(note && NOTES){
    const last=segs[segs.length-1];
    const at=last?px(last[1]+last[2])+9:LBL;
    lo+=`<text x="${at.toFixed(1)}" y="${y+2.5}" ${MONO} font-size="6.5" fill="${C.mut}">${note}</text>`;
  }
  const body=(dim===1?lo:`<g opacity="${dim}">${lo}</g>`)+(hi?`<g>${hi}</g>`:'');
  return s+body+hit;
}

/**
 * An axis under the plot. Ticks come from the scale that placed the bars, so a
 * label can only ever name a position the drawing actually reaches.
 *
 * Two tiers: the coarse one carries the date or hour a long run crosses, the
 * fine one carries offsets within it, so a run measured in days keeps its
 * resolution without inventing a second axis.
 */
export function axis(y, ticks, {tier2=[], brk=[], top=0, label}={}){
  let s='';
  /**
   * A break is a compression of the AXIS, not a hole in the data — there is
   * almost always a row running straight through it, usually a step.sleep()
   * spanning the whole stretch. Filling the band with hatch said "nothing
   * happened here" and beat the bar that was there, which is the opposite of
   * the truth.
   *
   * So the break is drawn on the axis itself and nowhere else: the rule stops,
   * a pair of dashed uprights fence the compressed stretch, and the elapsed
   * time is named between them. The bars keep their own substance and simply
   * get narrower, which is what compression actually did to them.
   */
  const cuts=[];
  for(const b of brk) cuts.push([px(b[0]),px(b[1]),b[2]]);
  // The rule is drawn in the segments between cuts, so the axis visibly stops.
  let at=LBL;
  for(const [x0,x1] of cuts){
    if(x0>at) s+=`<line x1="${at}" y1="${y}" x2="${x0}" y2="${y}" stroke="${C.idle}" stroke-width="1"/>`;
    at=x1;
  }
  s+=`<line x1="${at}" y1="${y}" x2="${LBL+PLOT}" y2="${y}" stroke="${C.idle}" stroke-width="1"/>`;
  for(const [x0,x1,t] of cuts){
    for(const x of [x0,x1])
      s+=`<line x1="${x}" y1="${top?top:y-4}" x2="${x}" y2="${y+4}" stroke="${C.mut}" stroke-width="1" stroke-dasharray="2 2.5" opacity=".8"/>`;
    if(t) s+=`<text x="${((x0+x1)/2).toFixed(1)}" y="${y-4}" ${MONO} font-size="6" fill="${C.mut}" text-anchor="middle">${t}</text>`;
  }
  for(const [p,t] of ticks)
    s+=`<line x1="${px(p)}" y1="${y}" x2="${px(p)}" y2="${y+3}" stroke="${C.idle}" stroke-width="1"/>`+
       tag(px(p), y+11, t);
  for(const [p,t] of tier2)
    s+=tag(px(p), y+21, t, C.ink2);
  return s;
}

/**
 * A collapsed group: one row standing in for many that share a shape. It draws
 * the envelope the members occupied and a tick per member, so the count and
 * the spread are both readable without expanding it.
 */
export function groupRow(i, {n, x, w, members, kind='good', note=''}){
  const y=cy(i);
  let s=tag(2, y+2.5, n, C.mut);
  s+=`<rect x="${px(x)}" y="${y-5}" width="${(w/100)*PLOT}" height="10" rx="2" fill="var(--surface-2)" stroke="${C.idle}" stroke-width="1"/>`;
  members.forEach(mm=>{
    s+=`<rect x="${px(mm[0])}" y="${y-3.5}" width="${Math.max(1.2,(mm[1]/100)*PLOT)}" height="7" rx="1" fill="url(#hx-${mm[2]||kind})"/>`;
  });
  if(note) s+=tag(px(x+w)+8, y+2.5, note);
  return s;
}

/**
 * The surrounding trace: the minimap, the Run row above, finalization below.
 *
 * A figure without them is a diagram, and a diagram lets a rule pass that a
 * real trace would break — most obviously the minimap, which the vocabulary
 * describes but which no figure ever drew *in place*, so nothing established
 * where it sits relative to the Run row and the steps.
 *
 * It is dimmed **and blurred**. Dimming alone still invites reading, and these
 * rows are meant to be present and plausible rather than legible: the eye
 * should land on the figure's own rows and take the rest as context.
 *
 * One frame row sits above the figure (Run) and one below (Finalization). A
 * figure that draws its own Run row gets no slot reserved for the suppressed
 * one, or it renders with a blank row at the top.
 */
export const FRAME_ROWS_ABOVE=1;
const CTX_O=0.34;

/**
 * Scenario figures are framed; reference figures (the vocabulary panel, the
 * primer, the detail views) are not, because they are showing a primitive
 * rather than a run. A generator opts its whole file in with `setFrame(true)`
 * rather than every call site passing it — 37 call sites with four different
 * signatures is how a scripted edit silently misses one.
 */
let FRAMED=false;
// DS_FRAME=0 builds the figures unframed, which is how they are exported for the
// user-facing docs: the blurred surround is a design-review device for us, and
// in docs it reads as something being hidden from the reader.
export const setFrame=on=>{FRAMED=process.env.DS_FRAME==='0'?false:on;};

export function traceFrame(rows,k,{end,hasOwnRun,pad=0,breaks=[],running=false}){
  const above=hasOwnRun?0:FRAME_ROWS_ABOVE;
  // Finalization sits a full row below whatever the last row was, which may
  // have been on the tighter span pitch.
  const ys=rowYs(rows);
  const finY=(ys.length?ys[ys.length-1]:cy(0)-ROW)+ROW+ROW*above;

  // Finalization is a discovery request like any other — it asks the SDK what
  // is next and the answer is "nothing". So it is queued, it waits, it starts,
  // and the bar is `disc`: **your app executes for it**, and you are billed for
  // it. Computed before the Run row, because the Run row has to contain it.
  const fx=Math.min(end+2,90), fq=2.2, fw=4.5, finEnd=running?end:fx+fq+fw;

  /**
   * The Run row is the whole overview, and it is **derived from every row in
   * the trace, finalization included**. It was built from the figure's own rows
   * only, so the run appeared to do nothing during finalization and stopped
   * short of where the trace actually ended — the profile disagreeing with the
   * rows underneath it, which is the one thing it must never do.
   *
   * It replaced a separate minimap that drew the same run in the same place in
   * the same colours; the only way to tell them apart was to make one
   * deliberately thinner, which was treating a symptom. One overview cannot
   * disagree with itself.
   */
  const intervals=[];
  const add=(kd,x,w)=>{ if(COMPUTE.has(base(kd))) intervals.push({a:x,b:x+w,ok:!isFail(kd)}); };
  rows.forEach(r=>(r.segs||[]).forEach(([kd,x,w])=>add(kd,x,w)));
  if(!running) add('disc',fx+fq,fw);

  // The run resolves as its LAST interval, not as "did anything fail". A run
  // that threw and then succeeded is a recovery, and resolving it red would say
  // the opposite of what the row underneath it shows.
  const last=intervals.reduce((m,v)=>(!m||v.b>m.b)?v:m,null);
  const run=hasOwnRun?'':runProfile(0,{to:finEnd,intervals,breaks,
    resolved:running?null:((last&&last.ok===false)?EV.failed:EV.ok)},k);

  // A run still going has not been finalized. Drawing the row anyway would be
  // the frame asserting an event that has not happened.
  const fin=running?'':tag(2,finY+2.5,'Finalization')+
    barSvg('idle',fx,fq,finY,{k:1,floor:k,o:1})+
    barSvg('disc',fx+fq,fw,finY,{k:1,floor:k,o:1})+
    dot(px(fx),finY,EV.queued)+
    dot(px(fx+fq),finY,EV.started)+
    dot(px(finEnd),finY,EV.ok);

  // The frame shares the row grid with the figure, and the inner content keeps
  // its own `cy` attributes because it is translated as a group — so the Run
  // row and the figure rows read as one row to anything parsing the SVG back.
  // Tag the frame marks so the validator skips them: context, not rows.
  const tagCtx=t=>t.replace(/<circle class="ev /g,'<circle class="ev ctx ');
  // Two layers, both currently soft. Kept split because the Run row is the one
  // piece of the surround that is sometimes the subject rather than context.
  return {sharp:tagCtx(run), soft:tagCtx(fin)};
}

/**
 * The elastic axis, computed rather than placed by hand.
 *
 * Two rules, and the second is the one that makes the first safe:
 *
 * 1. **Dead time is worth almost none of the width.** Any stretch with nothing
 *    executing, longer than `threshold` of the run, collapses to a band.
 *
 * 2. **Everything that survives shares ONE scale.** The width left over is
 *    divided between the live stretches in proportion to their real duration —
 *    30s on the left and 10s on the right get 75% and 25% of it. Which is the
 *    same thing as saying a second is worth the same number of pixels wherever
 *    it lands, so two spans in different parts of the run are still honestly
 *    comparable. Without this, compressing a gap would silently rescale one
 *    half of the trace against the other.
 *
 * Rule 2 is why N compressions need no special case: the arithmetic is total
 * live width over total live time, applied everywhere.
 *
 * **The bands share a budget.** All of them together get at most `budget` of the
 * plot, so a trace with twenty idle stretches does not spend its whole width on
 * the parts where nothing happened — each band just gets thinner, down to a
 * single marked line. A band never grows to fit its content, because its
 * content is precisely the thing not worth space.
 *
 * Returns `at(t)` — real time to drawn position — and the bands to mark.
 */
/**
 * The stretches of a run where **nothing at all was executing**.
 *
 * This is what decides where compression is allowed. A gap in one row is not
 * dead time — another row may be working through it — so the compute intervals
 * of every row are merged first and the holes in that union are the candidates.
 * Any other rule would compress a stretch that something was running in.
 */
export function deadStretches(total, compute=[]){
  const busy=[...compute].filter(([a,b])=>b>a).sort((x,y)=>x[0]-y[0]);
  const merged=[];
  for(const [a,b] of busy){
    const last=merged[merged.length-1];
    if(last && a<=last[1]) last[1]=Math.max(last[1],b);
    else merged.push([a,b]);
  }
  const gaps=[]; let at=0;
  for(const [a,b] of merged){ if(a>at) gaps.push([at,a]); at=Math.max(at,b); }
  if(at<total) gaps.push([at,total]);
  return gaps;
}

export function elastic(total, dead=[], {plot=86, threshold=0.03, budget=16, minBand=1.1, maxBand=4, inset=2, compute}={}){
  // Given the compute intervals, work out the dead stretches rather than being
  // told them: only a stretch with nothing running anywhere may be compressed.
  if(compute) dead=deadStretches(total, compute);
  let cuts=dead
    .map(([a,b])=>[Math.max(0,a),Math.min(total,b)])
    .filter(([a,b])=>b-a > total*threshold)
    .sort((x,y)=>x[0]-y[0]);
  const given=cuts.slice();

  // The cut takes the MIDDLE of a dead stretch, leaving its ends drawn to
  // scale, so the bar visibly begins, gets torn, and resumes. Cutting at the
  // edges of the stretch says only that something happened between two rows;
  // cutting inside it says which bar was compressed. Two passes, because how
  // much real time `inset` drawn units buys depends on the scale the cuts set.
  for(let pass=0; pass<2 && cuts.length; pass++){
    const w = Math.max(minBand, Math.min(maxBand, budget/cuts.length));
    const live = total - cuts.reduce((n,[x,y])=>n+(y-x), 0);
    const per = live>0 ? (plot - w*cuts.length)/live : 0;
    if(!per) break;
    const grab = inset/per;
    cuts = given.map(([oa,ob])=>{
      const g=Math.min(grab, Math.max(0,(ob-oa)/2 - 1));
      return [oa+g, ob-g];
    }).filter(([x,y])=>y>x);
  }

  const bandW = cuts.length ? Math.max(minBand, Math.min(maxBand, budget/cuts.length)) : 0;
  const liveTime = total - cuts.reduce((n,[a,b])=>n+(b-a), 0);
  const perUnit = liveTime>0 ? (plot - bandW*cuts.length)/liveTime : 0;

  const at=t=>{
    let x=0, seen=0;
    for(const [a,b] of cuts){
      if(t<=a) break;
      x += Math.max(0,(Math.min(t,a)-seen))*perUnit;
      seen = Math.min(t,a);
      if(t<=b){ return x + bandW*((t-a)/(b-a)); }   // inside the band
      x += bandW; seen = b;
    }
    return x + (t-seen)*perUnit;
  };

  // A band too narrow to hold them drops its label, then its tear. The rules
  // and the blur always stay: they are what says "not to scale".
  const marks = cuts.map(([a,b])=>({
    p0: at(a), p1: at(b),
    label: bandW>=2.6, tear: bandW>=1.8,
  }));
  return {at, bands:marks, bandW, cuts};
}

/**
 * A compressed stretch of dead time.
 *
 * The rule is simple and the threshold is low: **if nothing is executing for
 * more than a few percent of the run, that stretch is worth almost none of the
 * width.** It collapses to a fixed narrow band whatever it actually was — an
 * hour and seven days get the same few pixels, because the point is that the
 * space belongs to the work instead.
 *
 * Three cues, because one is not enough to overcome how strongly a time axis
 * reads as linear:
 *
 *   - a **torn-page zigzag** down the middle of the band, the same idiom the
 *     fixture scrubber uses;
 *   - **full-height rules** at both edges, so the cut crosses every row rather
 *     than being a mark on the axis that rows quietly ignore;
 *   - **a blur of whatever passes through the band**, which says "this width is
 *     not to scale" without hiding that a row is running through it.
 *
 * Only the drawing compresses. Every reported duration is still wall clock.
 */
export function compression(breaks, h, uid){
  if(!breaks || !breaks.length) return {clip:'', over:''};
  // A wide band can take a soft blur; eight narrow ones cannot — at full
  // strength they read as a row of smears cutting the trace up rather than as
  // one region being marked as not-to-scale.
  const n=breaks.length;
  const sd=Math.max(0.3, 1.4 - (n-1)*0.16);
  // Two full-height lines mark a cut. Sixteen of them at full strength cut the
  // trace into ribbons and become the loudest thing in it, so they fade as they
  // multiply — the same reasoning as the blur above.
  const ruleO=Math.max(0.28, 0.9 - (n-1)*0.09).toFixed(2);
  const filter=`<filter id="cmp-b-${uid}" x="-30%" y="-10%" width="160%" height="120%">`+
    `<feGaussianBlur stdDeviation="${sd.toFixed(2)}"/><feColorMatrix type="saturate" values="0.55"/></filter>`;
  const rects=breaks.map(([a,b])=>
    `<rect x="${px(a).toFixed(1)}" y="0" width="${(px(b)-px(a)).toFixed(1)}" height="${h}"/>`).join('');
  const clip=`<defs>${filter}<clipPath id="cmp-${uid}">${rects}</clipPath></defs>`;
  const over=breaks.map(([a,b,t])=>{
    const x0=px(a), x1=px(b), mid=(x0+x1)/2;
    return `<rect x="${x0.toFixed(1)}" y="0" width="${(x1-x0).toFixed(1)}" height="${h}" fill="var(--ground)" opacity=".28"/>`+
      [x0,x1].map(x=>`<line x1="${x.toFixed(1)}" y1="0" x2="${x.toFixed(1)}" y2="${h}" stroke="${C.idle}" stroke-width="1" opacity="${ruleO}"/>`).join('')+
      (t?`<text x="${mid.toFixed(1)}" y="${(h/2+2).toFixed(1)}" ${MONO} font-size="6.5" `+
         `fill="${C.ink2}" text-anchor="middle" paint-order="stroke" `+
         `stroke="var(--ground)" stroke-width="2.6" stroke-linejoin="round">${t}</text>`:'');
  }).join('');
  return {clip, over};
}

let UID=0;

export function fig(rows,extra='',label='',under='',opts={}){
  if(under&&typeof under==='object'){ opts=under; under=''; }
  if(opts.rib){
    under=ribbon(opts.rib.x,opts.rib.rows.map(i=>cy(i)))+under;
    rows=rows.map((r,i)=>opts.rib.rows.includes(i)?{...r,noHalo:[opts.rib.x]}:r);
  }
  const framed=opts.frame!==undefined?!!opts.frame:FRAMED;
  const ownRun=rows.some(r=>r.run||r.n==='Run');
  const above=framed?(ownRun?0:FRAME_ROWS_ABOVE):0;
  const n=opts.rowCount||rows.length;
  const lastY=rows.length?rowYs(rows)[rows.length-1]:cy(0)-ROW;
  const h=lastY+ROW*(1+above+(framed?1:0))+TOP+(opts.pad||0)
    -(rows.length?0:ROW);
  const ends=rows.flatMap(r=>(r.segs||[]).map(([,x,w])=>x+w));
  const max=ends.length?Math.max(...ends):100;
  const k=opts.scale||(max>0?Math.min(86/max,3):1);
  // Rows are placed from their own pitches, so a block of span rows packs
  // tighter than the trace around it.
  const ys=rowYs(rows);
  const body=under+rows.map((r,i)=>row(i,r,k,ys[i])).join('')+extra;
  const M=opts.margin||0;
  // Annotations are placed after the stretch, in final coordinates, so a
  // leader lands on the bar it points at rather than being scaled off it.
  const XS=p=>LBL+(px(p)-LBL)*k;
  const DY=ROW*above;
  const over=opts.over?opts.over(XS,i=>cy(i)+DY):'';
  const inner=framed
    ? `<g transform="translate(0,${DY})">${stretch(body,LBL,k)}</g>`
    : stretch(body,LBL,k);
  let ctx='';
  if(framed){
    const F=traceFrame(rows,k,{end:max,hasOwnRun:ownRun,pad:opts.pad||0,breaks:opts.breaks||[],running:!!opts.running});
    const soft=`<g filter="url(#ctxblur)" opacity="${CTX_O}">${stretch(F.soft,LBL,k)}</g>`;
    ctx=(opts.breaks&&opts.breaks.length)
      ? stretch(F.sharp,LBL,k)+soft
      : `<g filter="url(#ctxblur)" opacity="${CTX_O}">${stretch(F.sharp,LBL,k)}</g>`+soft;
  }
  // The compressed band blurs whatever runs through it. Done by drawing the
  // whole figure a second time, clipped to the band and filtered — SVG has no
  // backdrop-filter, and a flat scrim would hide the row rather than soften it.
  const cmp=compression(opts.breaks, h, ++UID);
  // Only the figure's own rows are blurred. The Run row is deliberately left
  // sharp: its torn track is what says "compressed here", and blurring the one
  // element carrying that message defeats drawing it at all.
  const blurred=cmp.clip
    ? `<g clip-path="url(#cmp-${UID})" filter="url(#cmp-b-${UID})">${inner}</g>`
    : '';
  return `<svg viewBox="${-M} 0 ${W+M*2} ${h}" role="img" aria-label="${label}">`+
    HATCH+BLURDEF+cmp.clip+ctx+inner+blurred+cmp.over+over+`</svg>`;
}

const BLURDEF=`<defs><filter id="cmpblur" x="-30%" y="-10%" width="160%" height="120%"><feGaussianBlur stdDeviation="1.4"/><feColorMatrix type="saturate" values="0.55"/></filter><filter id="ctxblur" x="-4%" y="-30%" width="108%" height="160%">`+
  `<feGaussianBlur stdDeviation="0.62"/>`+
  `<feColorMatrix type="saturate" values="0.25"/>`+
  `</filter></defs>`;

/**
 * The tie between one discovery request and every step it queued.
 *
 * A tapered ribbon threaded down the members' queued circles: it swells to just
 * wider than a circle where one sits, and pinches between them. Drawn under the
 * rows, so each circle reads as sitting ON it rather than beside it — the
 * grouping costs no row and favours no member.
 */
export function ribbon(xp, ys, {o=1, w=3.4}={}){
  if(ys.length<2) return '';
  const x=px(xp);
  return `<rect x="${x-w/2}" y="${ys[0]}" width="${w}" height="${ys[ys.length-1]-ys[0]}" fill="${C.disc}" opacity="${0.95*o}"/>`;
}
