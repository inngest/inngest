import * as R from './rules.mjs';
import { GEOM } from './vocabulary.mjs';
export const {W,LBL,RGT,PLOT,ROW,TOP}=GEOM;
export { GEOM };
export { C, EV, HATCH, dot, base, isFail, FILL, COMPUTE, NOCOMPUTE, ACTIVE, OPEN, paint, autoDots, discovered, lineage, barSvg, markSvg, cyOf, pxOf, BAR_INFO, EVENT_INFO, EVC, SUBSTANCE, substanceCSS, eventCSS } from './vocabulary.mjs';
import { C, EV, HATCH, dot, base, paint, autoDots, lineage, OPEN, COMPUTE, isFail, barSvg, causalRibbon as _cr, wire, fillGaps, stretch, remapX, BAR_INFO, EVENT_INFO } from './vocabulary.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";
let NOTES=false;
export const setNotes=on=>{NOTES=on;};
export const px=p=>LBL+(p/100)*PLOT;
export const cy=i=>TOP+R.GEOM.ROW_TOP+ROW*i;
/** Row pitch for a userland span: tight enough that the bars nearly touch. */
export const SPAN_ROW=R.GEOM.SPAN_ROW;
/** y for each row, honouring any that sit on the tighter span pitch. */
export const rowYs=rows=>{
  const ys=[]; let y=cy(0);
  rows.forEach((r,i)=>{ if(i) y+=rows[i].span?SPAN_ROW
    :(rows[i-1].span?SPAN_ROW+R.GEOM.SPAN_EXIT:ROW); ys.push(y); });
  return ys;
};
/** How many gaps of each pitch sit above row i — the multipliers CSS needs. */
export const rowSteps=rows=>{
  const out=[]; let n=0, m=0;
  rows.forEach((r,i)=>{ if(i){ if(rows[i].span||rows[i-1].span) m++; else n++; } out.push([n,m]); });
  return out;
};
export const arrow=(x1,y1,x2,y2,o=1)=>{
  const d=Math.max(10,Math.abs(x2-x1)*0.5);
  return `<path d="M${x1} ${y1} C ${x1+d} ${y1}, ${x2-d} ${y2}, ${x2-3} ${y2}" fill="none" stroke="${C.acc}" stroke-width="1.3" opacity="${o}"/>`+
    `<path d="M${x2-3} ${y2-2.2} L${x2} ${y2} L${x2-3} ${y2+2.2} Z" fill="${C.acc}" opacity="${o}"/>`;
};
export { wire };
export const causal=(sources,target,opts={})=>_cr(sources,target,{px,cy,...opts});
/**
 * The finalization request, spelled out by the figure that had one.
 *
 * A run that finished has one -- the request that asks what runs next and is
 * told nothing -- and it is a row like any other: queued at `t`, executing `q`
 * later, done `w` after that. Green, because it is your app executing: it is a
 * discovery request at the start and is retried like one, but once it succeeds
 * it will never plan.
 *
 * This exists so the three moments read the same way in every figure, NOT so
 * the drawing can supply one. The frame used to invent a finalization for any
 * run that had not declared one, which put a real part of the trace into
 * figures whose run had not got that far. A finalization happened or it did
 * not, and only the figure knows which.
 */
export const fin=(t,q,w)=>({n:'Finalization', kind:'step',
  at:[['queued',t],['started',t+q],['ok',t+q+w]]});

export const tag=(x,y,t,c=C.mut)=>`<text x="${x}" y="${y}" ${MONO} font-size="6.5" fill="${c}">${t}</text>`;

/**
 * The label column and the line that closes it.
 *
 * Every row name in every figure goes through here, so the column is one width
 * everywhere and the bars all start at the same x -- which is the point of
 * giving it a fixed width at all. A name too long for the column is elided;
 * the whole name is still in `data-n`.
 */
export const RULE_X=LBL-R.GEOM.LBL_RULE;
export function rowLabel(y,n,{span=false,o=1}={}){
  const x=R.GEOM.LBL_X+(span?R.GEOM.LBL_INDENT:0);
  const f=span?R.GEOM.LBL_SPAN_FONT:R.GEOM.LBL_FONT;
  const t=R.elide(String(n), RULE_X-x-R.GEOM.LBL_PAD, f);
  // Centred on the row from the font size, so shrinking the label does not
  // leave it sitting off the line its bar is on.
  return `<text class="rowlbl" data-n="${n}" x="${x}" y="${(y+f*0.36).toFixed(1)}" ${MONO} `+
    `font-size="${f}" fill="${C.mut}" opacity="${o}">${t}</text>`;
}
/** The divider, full height, so the column reads as a column and not as an indent. */
export const gutter=h=>`<rect class="gut" x="${(RULE_X-0.3).toFixed(1)}" y="0" width="0.6" `+
  `height="${h}" fill="${C.idle}" opacity=".5"/>`;

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
export function runProfile(i,{to=100,intervals=[],resolved,n='Run',breaks=[],lead='',opened='queued'},sc=1){
  // The queue the run opened with is named here rather than drawn, so it does
  // not push every row's work to the right of a gap that says the same thing
  // about all of them.
  if(lead) n=n+' (+'+lead+')';
  const y=cy(i);
  // Colour is a priority, not an average: see RUN_RANK. A slice carries the
  // rank of the bar it came from, and where slices overlap the higher rank is
  // drawn last and so is the one left showing.
  const col=v=> R.RUN_COLOUR[v.rank!=null?v.rank:2];
  let s=rowLabel(y,n);
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
    const [lo,hi]=R.BAND.tearZigs;
    const n=Math.min(hi,Math.max(lo,Math.round((x1-x0)/8))), step=(x1-x0)/n, amp=R.BAND.tearAmp;
    let d=`M${x0.toFixed(1)} ${y}`;
    for(let j=0;j<n;j++)
      d+=` L${(x0+step*(j+0.5)).toFixed(1)} ${(y+(j%2?amp:-amp)).toFixed(1)}`+
         ` L${(x0+step*(j+1)).toFixed(1)} ${y}`;
    return `<path d="${d}" fill="none" stroke="${C.idle}" stroke-width="${R.BAND.tearWidth}" stroke-linecap="round" stroke-linejoin="round"/>`;
  };
  {
    let at=LBL;
    for(const [a,b] of breaks){
      const x0=Math.max(LBL,px(a)), x1=Math.min(end,px(b));
      if(x1<=x0) continue;
      if(x0>at) s+=`<rect class="run-track" x="${at.toFixed(1)}" y="${y}" width="${(x0-at).toFixed(1)}" height="5" rx="1.2" fill="${C.idle}"/>`;
      s+=tear(x0,x1); at=x1;
    }
    if(end>at) s+=`<rect class="run-track" x="${at.toFixed(1)}" y="${y}" width="${(end-at).toFixed(1)}" height="5" rx="1.2" fill="${C.idle}"/>`;
  }
  // Drawn worst-last, so where slices overlap the higher rank survives.
  const ordered=[...intervals].sort((p,q)=>(p.rank||0)-(q.rank||0));
  for(const v of ordered){
    const a=v.a, b=Math.min(v.b,to);
    if(b<=a) continue;
    // Failure stays findable even when the interval is tiny.
    const w=Math.max((v.rank===3?R.GEOM.MIN_FAIL_W:R.GEOM.MIN_W)/sc, ((b-a)/100)*PLOT);
    s+=`<rect class="run-slice" x="${px(a).toFixed(1)}" y="${y}" width="${w.toFixed(1)}" height="8" rx="1.2" fill="${col(v)}"/>`;
  }
  // The Run row opens on the moment the run was actually in when the drawing
  // starts. Trim its opening queue away and the drawing starts with the run
  // already executing, which is what its first step says too.
  s+=dot(LBL,y,opened==='started'?EV.started:EV.queued);
  if(resolved) s+=dot(LBL+(to/100)*PLOT,y,resolved);
  return s;
}

export function row(i,r,sc=1,yy){
  if(r.run) return runProfile(i,r,sc);
  const {n,segs:rawSegs=[],dim=1,dots,note,sel,noHalo,lit,litDots}=r;
  /**
   * Two rows exist that are deliberately half a row.
   *
   * `marks:false` is a row with no lifecycle to report — a network phase inside
   * a request is a duration, not something that is enqueued, starts and
   * resolves. `bars:false` is the reverse, and there is one: the figure whose
   * entire point is that a span is three timestamps before it is anything else.
   */
  const noMarks=r.marks===false, noBars=r.bars===false;
  const segs=fillGaps(rawSegs.map(([k,x,w,ms])=>({kind:k,x,w,ms})))
    .map(g=>[g.kind,g.x,g.w,g.ms]);
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
  // stretch() is a no-op at or below 1, so only pre-divide when it will
  // actually scale back up — dividing unconditionally made the band WIDER on
  // every figure whose content runs past the fit width.
  const sx = sc > 1.001 ? sc : 1;
  const litBar=(k,x)=>!lit ? dim
    : (lit.some(v=>typeof v==='string' ? v===base(k) : Math.abs(v-x)<0.01) ? 1 : dim);
  const litDot=p=>!litDots ? dim : (litDots.some(v=>Math.abs(v-p)<0.01) ? 1 : dim);
  if(sel) s+=`<rect class="selband${r.span?' sp':''}" x="${(LBL-3/sx).toFixed(2)}" y="${y}" `+
    `width="${((PLOT+6)/sx).toFixed(2)}" height="15" fill="${C.acc}" opacity=".13" rx="2"/>`;
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
  // Marks come from the row's moments where it has them, and are read back off
  // the bars only for the handful of rows still drawn without any.
  const auto=noMarks ? []
    : r.at&&r.at.length ? R.marks(r.at)
    : autoDots(segs.map(([k,a,w])=>({kind:k,x:a,w})));
  let lo='', hi='';
  // The dim layer is the WHOLE row, not the leftover after the lit parts are
  // taken out. Splitting it that way separated bars from marks into different
  // groups, so a lit bar was drawn OVER a dim mark that belongs on top of it.
  // Drawing the complete row at `dim` and then repainting only the lit parts
  // over it keeps bars under marks inside both layers.
  const put=(on,frag)=>{ lo+=frag; if(on) hi+=frag; };
  if(r.spans) lo+=[0,1,2].map(j=>
    `<rect x="0.5" y="${(y-2.6+j*2.2).toFixed(1)}" width="${R.GEOM.LBL_GLYPH}" height="1" rx="0.5" fill="${C.mut}" opacity="${0.85-j*0.22}"/>`).join('');
  lo+=rowLabel(y,n,{span:!!r.span, o:r.span?0.78:1});
  // A userland span is a subdivision of the step above it, not a peer, so it is
  // drawn thinner. Nesting and weight carry that, not a new colour: an OTel span
  // IS your code, so it keeps the same status colours the step has.
  const bh=(r.thin||r.span)?GEOM.BAR_H*0.6:GEOM.BAR_H;
  if(!noBars) segs.forEach(([k,a,w])=>{ put(litBar(k,a)===1, barSvg(k,a,w,y,{k:1,floor:sc,h:bh})); });
  if(!r.span) (dots||auto).forEach(d=>{
    const onRib=(noHalo||[]).some(p=>Math.abs(p-d.p)<0.01);
    // How we came to know a moment is a fact about the moment, so it is
    // carried on the row as the instants that arrived by checkpoint.
    const cp=(r.cp||[]).some(p=>Math.abs(p-d.p)<0.01);
    put(litDot(d.p)===1, dot(px(d.p),y,onRib?'ribbon':(d.c||C.mut),1,R.GEOM.MARK_R,true,cp,d.s||''));
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
      const b=BAR_INFO[base(g[0])];
      if(b) parts.push({t:'b',k:base(g[0]),n:b[0],d:b[1],
                        ms:g[3]!=null?R.human(g[3]):undefined});
    });
    for(;mi<marks.length;mi++){ const e=EVENT_INFO[marks[mi].c]; if(e) parts.push({t:'e',k:marks[mi].c,n:e[0],d:e[1]}); }
    if(parts.length) hit=`<rect class="rowhit" x="${(LBL-6/sx).toFixed(2)}" y="${y}" width="${((PLOT+12)/sx).toFixed(2)}" height="16" fill="transparent" data-row="${n}" data-parts='${JSON.stringify(parts).replace(/'/g,"&apos;")}'/>`;
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
  return `<g class="r" style="--i:${i};--s:0">${groupRowAt(i,{n,x,w,members,kind,note})}</g>`;
}
function groupRowAt(i, {n, x, w, members, kind='good', note=''}){
  const y=cy(i);
  let s=rowLabel(y,n);
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
export const setFrame=on=>{FRAMED=R.ENV.DS_FRAME==='0'?false:on;};

/**
 * The surround: the Run row, and nothing else.
 *
 * It used to draw the finalization too -- and, worse, INVENT one for any run
 * that had not recorded a finalization, at coordinates it worked out itself. A
 * finalization is a real request that either happened or did not, so a figure
 * declares it as moments like every other row and the frame never supplies it.
 * The frame is left with the one thing that genuinely is not a row.
 */
export function traceFrame(rows,k,{hasOwnRun,breaks=[],running=false,lead='',trimmed=false}){
  /**
   * The Run row is the whole overview, and it is **derived from every row in
   * the trace, finalization included**. It was built from the figure's own rows
   * only, so the run appeared to do nothing during finalization and stopped
   * short of where the trace actually ended -- the profile disagreeing with the
   * rows underneath it, which is the one thing it must never do.
   *
   * It replaced a separate minimap that drew the same run in the same place in
   * the same colours; the only way to tell them apart was to make one
   * deliberately thinner, which was treating a symptom. One overview cannot
   * disagree with itself.
   */
  const intervals=[];
  const add=(kd,x,w)=>{ const rank=R.runRank(kd); if(rank!=null) intervals.push({a:x,b:x+w,rank}); };
  rows.forEach(r=>(r.segs||[]).forEach(([kd,x,w])=>add(kd,x,w)));
  // The run resolves as its LAST interval, not as "did anything fail". A run
  // that threw and then succeeded is a recovery, and resolving it red would say
  // the opposite of what the row underneath it shows.
  const last=intervals.reduce((m,v)=>(!m||v.b>m.b)?v:m,null);
  // To the end of the axis, because the axis IS the run: it runs from the
  // first moment in the trace to the last, and the finalization is the last.
  const run=hasOwnRun?'':runProfile(0,{to:100,intervals,breaks,lead,opened:trimmed?'started':'queued',
    resolved:running?null:((last&&last.rank===3)?EV.failed:EV.ok)},k);
  // The frame shares the row grid with the figure, and the inner content keeps
  // its own `cy` attributes because it is translated as a group -- so the Run
  // row and the figure rows read as one row to anything parsing the SVG back.
  // Tag the frame marks so the validator skips them: context, not rows.
  const tagCtx=t=>t.replace(/<circle class="ev /g,'<circle class="ev ctx ');
  return {sharp:tagCtx(run), soft:''};
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

export function elastic(total, dead=[], {plot=100, threshold=R.ELASTIC.floor,
    budget=R.ELASTIC.budget, minBand=R.ELASTIC.minBand, maxBand=R.ELASTIC.maxBand,
    inset=R.ELASTIC.inset, compute}={}){
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
    // How long this band actually stands for, in whatever unit the caller
    // measured in. A band names its OWN elapsed time; they used to share one
    // string handed down by the figure, so eight different gaps all read "2m".
    span: b-a,
    label: bandW>=R.ELASTIC.labelAt, tear: bandW>=R.ELASTIC.tearAt,
  }));
  return {at, bands:marks, bandW, cuts};
}

/**
 * Lay a run out from REAL DURATIONS. This is the one place the elastic rule
 * lives, and every figure that goes through it gets the rule for free.
 *
 * Figures used to carry percentages — "the sleep is 96% of the run" — and a
 * percentage cannot be compressed, because whether a stretch is worth the width
 * depends on how long it actually was. A 2s sleep next to 10ms of work and a 7d
 * sleep next to 10ms of work are the same percentage and want different
 * drawings. So a figure declares milliseconds and this decides the rest:
 *
 *   - the dead stretches are derived from where nothing was executing, across
 *     every row (`deadStretches`), never nominated;
 *   - anything over the threshold collapses to a band;
 *   - everything that survives shares one scale.
 *
 * Returns rows in drawn coordinates plus the breaks to mark, so a caller passes
 * the result straight to `fig`. Change the rule here and every figure moves.
 */
export function layout(total, rows, opts={}){
  /**
   * Rows may declare moments in real time instead of bars. They are the same
   * declaration a figure makes anywhere else -- `['started', 0], ['ok', 41]` --
   * only measured in milliseconds, so the elastic rule has durations to judge
   * rather than proportions.
   */
  rows=rows.map(r=>{
    if(r.run || !r.at || !r.at.length) return r;
    const segs=R.derive(r.kind||'step', r.at, r.end, {reported:!!r.reported});
    return {...r, segs:segs.map(([k,x,w])=>[k,x,x+w])};   // layout works in start/end
  });
  const compute=[];
  rows.forEach(r=>{
    if(r.run) return;                       // the Run row is derived, not input
    (r.segs||[]).forEach(([kd,a,b])=>{ if(COMPUTE.has(base(kd))) compute.push([a,b]); });
    // A collapsed group draws its members rather than its own bars, so without
    // this the rule reads the whole group as idle and compresses the work away.
    (r.group?r.group.members||[]:[]).forEach(([a,b,kd])=>{
      if(COMPUTE.has(base(kd||'good'))) compute.push([a,b]); });
  });
  // With compression off the axis is linear and there are no bands.
  /**
   * The same scale-free threshold fig() applies: a stretch is compressed when
   * it is longer than ALL the compute in the run put together, several times
   * over. Without it elastic falls back to "more than 3% of the run", which on
   * a 14ms run with 13ms of work compresses the 1ms tail -- a band standing for
   * a millisecond, in a figure whose point is that bands stand for days.
   */
  const busy=compute.reduce((n,[a,b])=>n+(b-a),0);
  // `linear` is a figure opting out: two of them exist to show the UNCOMPRESSED
  // proportion, so compressing them deletes their point.
  /**
   * A run with NO compute in it is never compressed.
   *
   * The threshold is a multiple of the run's total compute, and three times
   * nothing is nothing -- so every stretch qualified and a run whose whole
   * substance is a wait collapsed to a single 4-unit band with the rest of the
   * width empty. `step.sleep` on its own, a wait cancelled while open, a
   * waitForEvent that timed out: the figures that exist to show waiting were
   * exactly the ones the rule erased.
   *
   * The rule reads correctly the other way round: compression buys width for
   * the compute. Where there is none, there is nothing to buy it for, and the
   * wait is not dead time -- it is the whole trace.
   */
  const el=(R.FEAT.compress && !opts.linear && busy>0) ? elastic(total, [], {compute,
      threshold:Math.max(R.ELASTIC.floor, total>0?busy*R.ELASTIC.computeMultiple/total:0),
      ...opts})
    : {at:t=>t/total*(opts.plot||100), bands:[], cuts:[]};
  /**
   * A bar keeps its real duration as well as its drawn one.
   *
   * The drawn width is a fraction of an axis that may be compressed, so it
   * cannot be read back as a time -- and the popover has to name the interval,
   * not the number of pixels it got. Carried in the segment because that is
   * what survives every later pass; recovering it afterwards would mean
   * inverting the elastic axis, which is the arithmetic this is here to avoid
   * repeating.
   */
  const msPer = opts.unit==='s' ? 1000 : 1;
  const map=([kd,a,b])=>[kd, el.at(a), el.at(b)-el.at(a), (b-a)*msPer];
  const out=rows.map(r=>{
    if(r.run) return {...r,
      to: el.at(r.to!=null?r.to:total),
      intervals:(r.intervals||[]).map(v=>({...v, a:el.at(v.a), b:el.at(v.b)}))};
    return {...r,
      // A collapsed group's members are intervals in the same real time as
      // everything else, so they go through the axis with everything else.
      // They used to be mapped by the figure before it called fig(), which
      // baked the compression in and left this one figure unable to redraw.
      group:r.group?{...r.group,
        members:(r.group.members||[]).map(([a,b,k])=>[el.at(a), el.at(b)-el.at(a), k]),
        w:el.at(r.group.to!=null?r.group.to:total)}:r.group,
      segs:(r.segs||[]).map(map),
      at:(r.at||[]).map(mo=>mo.length===3?[mo[0],el.at(mo[1]),mo[2]]:[mo[0],el.at(mo[1])]),
      cp:r.cp?r.cp.map(el.at):r.cp,
      end:r.end!=null?el.at(r.end):r.end};
  });
  /**
   * Each band is named with the time IT compressed.
   *
   * A figure that measured in real durations can say how long a cut was; one
   * working in proportions cannot, and marks the cut without naming it. That is
   * the whole reason some bands carry a duration and some do not — not a rule
   * about which cuts deserve one.
   */
  const per = opts.unit==='s' ? 1000 : 1;
  const name = b => b.label ? R.human(b.span*per) : '';
  return {rows:out, breaks:el.bands.map(b=>[b.p0,b.p1,name(b)]), el};
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
  const n=breaks.length;   // how many cuts share the page

  // Two full-height lines mark a cut. Sixteen of them at full strength cut the
  // trace into ribbons and become the loudest thing in it, so they fade as they
  // multiply — the same reasoning as the blur above.
  const ruleO=Math.max(R.BAND.ruleFloor, R.BAND.ruleOpacity - (n-1)*R.BAND.ruleFalloff).toFixed(2);

  const rects=breaks.map(([a,b])=>
    `<rect class="cmpband" x="${px(a).toFixed(1)}" y="0" width="${(px(b)-px(a)).toFixed(1)}" height="${h}"/>`).join('');
  const clip='';
  const over=breaks.map(([a,b,t])=>{
    const x0=px(a), x1=px(b), mid=(x0+x1)/2;
    // The edges are rects rather than lines: a line's y2 is not a CSS
    // property, so it could not follow the pitch the way the band does.
    return `<rect class="cmpband" x="${x0.toFixed(1)}" y="0" width="${(x1-x0).toFixed(1)}" height="${h}" fill="var(--ground)" opacity=".28"/>`+
      [x0,x1].map(x=>`<rect class="cmpband" x="${(x-0.5).toFixed(1)}" y="0" width="1" height="${h}" fill="${C.idle}" opacity="${ruleO}"/>`).join('')+
      (t?`<g class="cmpmid"><text x="${mid.toFixed(1)}" y="${(h/2+2).toFixed(1)}" ${MONO} font-size="6.5" `+
         `fill="${C.ink2}" text-anchor="middle" paint-order="stroke" `+
         `stroke="var(--ground)" stroke-width="2.6" stroke-linejoin="round">${t}</text></g>`:'');
  }).join('');
  return {clip, over};
}

/**
 * Generated ids have to be unique across the PAGE, and each generator runs as
 * its own process with its own counter — so two of them both produced `cmp-5`
 * and a url(#cmp-5) resolved to whichever came first. Each generator gets a
 * range. Stable across the feature builds, so two identical drawings still
 * compare equal and ship once.
 */
let UID=+(R.ENV.DS_UID_BASE||0);

/**
 * Figures that contain a stretch the elastic rule would compress, but which
 * were not laid out by it — hand-placed geometry that the rule cannot reach.
 * The goal is for this list to be empty: a figure should be a list of events
 * and the drawing derived from them, so changing a rule changes every figure.
 */
export const UNRULED=[];
// Resolved only under Node: in a page `import.meta.url` is a data URL and
// there is no relative path to resolve against it.
const UNRULED_FILE=(typeof process!=='undefined' && process.on)
  ? new URL('./unruled.json',import.meta.url).pathname : null;
// Build-time bookkeeping. In a page there is nothing to write to and no build
// to fail, so it simply does not run.
if(typeof process!=='undefined' && process.on) process.on('exit',async()=>{
  if(!UNRULED.length) return;
  const fs=await import('fs');
  let prev=[]; try{ prev=JSON.parse(fs.readFileSync(UNRULED_FILE,'utf8')); }catch{}
  try{ fs.writeFileSync(UNRULED_FILE, JSON.stringify(prev.concat(UNRULED))); }catch{}
});

/**
 * Every row a figure declared, recorded so the moments->bars derivation can be
 * checked against what is actually authored. The goal is for the derivation to
 * reproduce all of them; the ones it cannot are the rows that still need to say
 * something the moments do not carry.
 */
export const AUTHORED=[];
const AUTHORED_FILE=(typeof process!=='undefined' && process.on)
  ? new URL('./authored.json',import.meta.url).pathname : null;
if(R.ENV.DS_AUDIT && AUTHORED_FILE) process.on('exit',async()=>{
  const fs=await import('fs');
  let prev=[]; try{ prev=JSON.parse(fs.readFileSync(AUTHORED_FILE,'utf8')); }catch{}
  try{ fs.writeFileSync(AUTHORED_FILE, JSON.stringify(prev.concat(AUTHORED))); }catch{}
});

/**
 * A row is a list of moments, and its bars are derived from them.
 *
 * A figure may declare `at` directly. One still written as `segs` is read back
 * into the moments it implies and then re-derived, so a figure authored the
 * older way goes through the same rule as one authored the new way — and
 * changing what a substance MEANS changes both. Bars that never passed through
 * `derive` were the whole problem: the rule existed and reached nothing.
 */
export function resolveRow(r){
  if(r.run) return r;
  if(r.at) return {...r, segs:R.derive(r.kind||'step', r.at, r.end, {reported:!!r.reported})};
  if(!r.segs || !r.segs.length) return r;
  const filled=fillGaps(r.segs.map(([k,x,w])=>({kind:k,x,w}))).map(g=>[g.kind,g.x,g.w]);
  const mo=R.moments(filled);
  // The moments stay on the row: what queued what is a fact about the run, and
  // everything drawn between rows reads it rather than being handed coordinates.
  return {...r, at:mo.at, kind:mo.kind, reported:mo.reported,
          segs:R.derive(mo.kind, mo.at, mo.end, {reported:mo.reported})};
}

let FIG_ID=+(R.ENV.DS_UID_BASE||0);
/** What every figure on the page was drawn from, in call order. */
export const FIGURES=[];
// Each generator is its own process, so each writes its own file and the page
// build reads them all back. Keyed by the id range that generator was given.
if(typeof process!=='undefined' && process.on) process.on('exit',async()=>{
  if(!FIGURES.length) return;
  const fs=await import('fs');
  const out=new URL('./figdata-'+(R.ENV.DS_UID_BASE||0)+'.json',import.meta.url).pathname;
  try{ fs.writeFileSync(out, JSON.stringify(FIGURES)); }catch{}
});

/**
 * The tick strip, drawn from the same axis the bars were laid out on.
 *
 * `at` is monotonic, so each drawn position is inverted back to the time it
 * stands for by bisection rather than by a second copy of the elastic
 * arithmetic. A tick landing inside a compressed band is dropped: the band is
 * the one part of the width that is not to scale, so a time written across it
 * would be the drawing contradicting itself.
 */
export function tickStrip(at, total, unit, k, breaks=[]){
  if(!(total>0)) return '';
  const inv=x=>{ let lo=0, hi=total;
    for(let i=0;i<48;i++){ const mid=(lo+hi)/2; if(at(mid)<x) lo=mid; else hi=mid; }
    return (lo+hi)/2; };
  const n=R.AXIS.ticks;
  // The tick marks sit at the bottom of the strip, the two label lines above.
  const yTick=-3, yLo=yTick-R.AXIS.tick-1.8, yHi=yLo-R.AXIS.line;
  const inBand=p=>(breaks||[]).some(([a,b])=>p>a+1e-9 && p<b-1e-9);
  let s=`<line x1="${LBL}" y1="${yTick}" x2="${(LBL+PLOT).toFixed(1)}" y2="${yTick}" stroke="${C.idle}" stroke-width="0.6" opacity=".45"/>`;
  for(let i=0;i<n;i++){
    const p=(i/(n-1))*100;
    if(inBand(p*k)) continue;
    const x=LBL+((p*k)/100)*PLOT;
    const [hi,lo]=R.tickLabel(unit==='s' ? inv(p)*1000 : inv(p));
    const anchor=i===0?'start':i===n-1?'end':'middle';
    const label=(t,yy,o)=>t?`<text x="${x.toFixed(1)}" y="${yy.toFixed(1)}" ${MONO} `+
      `font-size="${R.AXIS.font}" fill="${C.mut}" opacity="${o}" text-anchor="${anchor}">${t}</text>`:'';
    s+=`<line x1="${x.toFixed(1)}" y1="${(yTick-R.AXIS.tick).toFixed(1)}" x2="${x.toFixed(1)}" y2="${yTick}" stroke="${C.idle}" stroke-width="0.8" opacity=".7"/>`
      +label(hi,yHi,'.85')+label(lo,yLo,hi?'.6':'.85');
  }
  return s;
}

export function fig(rows,extra='',label='',under='',opts={}){
  // Kept before anything derives from them, so the record is what was asked for
  // rather than what it turned into.
  const rows0=rows, extra0=typeof extra==='string'?extra:'', under0=typeof under==='string'?under:'',
        opts0=(under&&typeof under==='object')?under:opts;
  rows=rows.map(resolveRow);
  /**
   * Rows measured in real time are laid out HERE.
   *
   * Callers used to run layout() themselves and hand over the result plus the
   * bands it produced, which baked the elastic rule into the figure: switching
   * compression off could not undo a band that had already been decided. Given
   * `ms`, fig() owns the whole path from events to drawing, so the same events
   * redraw differently when what a drawing means changes.
   */
  /**
   * Every figure is a run, measured.
   *
   * A figure declares moments and nothing else; where it does not also declare
   * how long the run took, the last moment IS how long it took. That makes the
   * numbers milliseconds rather than proportions, so the rules that need a
   * duration -- the scale-free compression threshold, the opening-queue trim --
   * reach every figure instead of only the ones that came from a capture.
   * Positions are unchanged where no rule fires: laying 0..86 out across the
   * axis puts it exactly where 0..86 was.
   */
  if(opts.ms==null && rows.some(r=>r.at&&r.at.length)){
    let end=0;
    for(const rw of rows){
      for(const [,x] of (rw.at||[])) end=Math.max(end,x);
      if(rw.end!=null) end=Math.max(end,rw.end);
    }
    if(end>0) opts={...opts, ms:end};
  }

  /**
   * The opening queue, hidden rather than drawn.
   *
   * Applies to any figure whose rows are moments, not just to the ones that
   * came from a captured run: hiding a run's first wait is a property of the
   * DRAWING, and a figure drawn from proportions has an opening wait too. Only
   * the label needs a duration, so a figure that never measured one is trimmed
   * without being told how much was taken.
   */
  let leadLabel='', leadTrimmed=false, axisAt=null, axisTotal=0;
  // Only where the figure is a RUN. A figure drawn from proportions has an
  // opening wait too, but several exist precisely to show it -- queue time,
  // flow control, finalization -- and trimming those leaves three identical
  // pictures where there were three different points. The feature hides a
  // real run's first wait; it is not a global eraser.
  if(R.FEAT.trim && opts.ms && opts.trimLead!==false){
    let first=Infinity;
    // Stops at the first thing that is not plain waiting. Being held by flow
    // control is queue time WITH A REASON, and hiding it would hide the reason.
    for(const rw of rows) for(const [k,x] of (rw.at||[]))
      if(k==='started' || k==='held'){ first=Math.min(first,x); break; }
    if(first>1e-9 && first<Infinity){
      leadTrimmed=true;
      if(opts.ms) leadLabel=R.human(opts.unit==='s'?first*1000:first);
      rows=rows.map(rw=>{
        const s=(rw.at||[]).map(mo=>mo.length===3?[mo[0],mo[1]-first,mo[2]]:[mo[0],mo[1]-first]);
        const kept=s.filter(mo=>mo[1]>=-1e-9);
        const open=(kept.length&&kept[0][1]<1e-9)?null:s.filter(mo=>mo[1]<0).pop();
        let at=(open?[[...open].map((v,q)=>q===1?0:v)]:[])
          .concat(kept.map(mo=>mo[1]<0?[...mo].map((v,q)=>q===1?0:v):mo));
        /**
         * A row whose ENTIRE wait was the wait being hidden opens already
         * running, so it loses the moment that named the wait.
         *
         * Only where the trim is what closed the gap: both moments landing on
         * zero is the trim having taken everything between them. A row queued
         * and started at the same instant later in the run still says it was
         * queued -- that instant is a fact about the run, not an artefact of
         * what this drawing is hiding.
         */
        if(at.length>1 && (at[0][0]==='queued'||at[0][0]==='held')
           && Math.abs(at[0][1])<1e-9 && Math.abs(at[1][1])<1e-9) at=at.slice(1);
        const cp0=(rw.cp||[]).map(t=>t-first).filter(t=>t>=-1e-9).map(t=>Math.max(0,t));
        return {...rw, cp:rw.cp?cp0:undefined, at,
          end:rw.end!=null?Math.max(0,rw.end-first):rw.end,
          segs:undefined};
      }).map(resolveRow);
      if(opts.ms) opts={...opts, ms:opts.ms-first};
    }
  }
  if(opts.ms){
    /**
     * The axis IS the run: from zero to the last moment in it.
     *
     * A capture's recorded duration can outlast its last span by a millisecond
     * or two of bookkeeping, and laying the axis out to THAT leaves the trace
     * ending short of the width with nothing drawn in the remainder -- an
     * unexplained gap, which is the one thing this drawing does not allow.
     */
    let last=0;
    rows.forEach(r=>{ (r.at||[]).forEach(([,t])=>{ last=Math.max(last,t); });
                      if(r.end!=null) last=Math.max(last,r.end); });
    if(last>0) opts={...opts, ms:last};
    const L=layout(opts.ms, rows, {plot:100, unit:opts.unit, linear:opts.linear});
    rows=L.rows; opts={...opts, breaks:L.breaks};
    // Kept so the tick strip reads the same axis the bars were laid on.
    axisAt=L.el.at; axisTotal=opts.ms;
  }
  /**
   * A run's opening queue time is named, not drawn. Done before the elastic
   * pass and before any scale is chosen, so everything downstream simply sees a
   * trace that begins when the work does.
   */
  if(under&&typeof under==='object'){ opts=under; under=''; }
  /**
   * The elastic rule, applied to every figure rather than to the ones that
   * remembered to ask.
   *
   * A figure's coordinates are already a linear time axis, and the threshold is
   * a FRACTION of the run — so percentages carry everything the rule needs and
   * no figure has to invent durations it never measured. What a percentage
   * cannot give is the band's label, since naming the elapsed time needs a
   * duration; a converted figure marks the cut without naming it.
   *
   * The pre-drawn content — arrows, ribbons, cables, annotations — is remapped
   * through the same function, which is what made this possible at all.
   */
  if(R.FEAT.compress && !opts.breaks && !opts.linear && rows.some(r=>r.segs)){
    const compute=[]; let end=0;
    rows.forEach(r=>(r.segs||[]).forEach(([kd,a,w])=>{
      end=Math.max(end,a+w);
      if(COMPUTE.has(base(kd))) compute.push([a,a+w]);
    }));
    (opts.busy||[]).forEach(([a,b])=>{ compute.push([a,b]); end=Math.max(end,b); });
    if(compute.length && end>0){
      /**
       * A dead stretch is only worth compressing when it DWARFS the work — not
       * merely when it is a few percent of the run. A flat fraction compressed
       * ordinary queue intervals, which are short, meaningful, and exactly the
       * thing the trace is there to show.
       *
       * The test is scale-free, which matters because a figure has proportions
       * and not durations: a stretch qualifies when it is longer than all the
       * compute in the run put together, several times over. Seven days beside
       * 100ms of work passes by a factor of millions; six units of queue beside
       * a hundred of work does not pass at all.
       */
      const busy=compute.reduce((n,[a,b])=>n+(b-a),0);
      const el=elastic(end, [], {compute, plot:end, inset:end*0.02,
        threshold:Math.max(R.ELASTIC.floor, busy*R.ELASTIC.computeMultiple/end)});
      if(el.bands.length){
        const at=el.at;
        rows=rows.map(r=>r.run
          ? {...r, to:r.to!=null?at(r.to):r.to, end:r.end!=null?at(r.end):r.end,
             intervals:(r.intervals||[]).map(v=>({...v,a:at(v.a),b:at(v.b)}))}
          : {...r, segs:(r.segs||[]).map(([k,a,w,ms])=>[k,at(a),at(a+w)-at(a),ms]),
             // The moments move with the bars. Marks are drawn from them now,
             // so leaving them behind put every circle at the position the
             // compression had just taken away.
             at:(r.at||[]).map(mo=>mo.length===3?[mo[0],at(mo[1]),mo[2]]:[mo[0],at(mo[1])]),
             dots:r.dots?r.dots.map(d=>({...d,p:at(d.p)})):r.dots,
             lit:r.lit?r.lit.map(v=>typeof v==='number'?at(v):v):r.lit,
             litDots:r.litDots?r.litDots.map(at):r.litDots,
             noHalo:r.noHalo?r.noHalo.map(at):r.noHalo,
             cp:r.cp?r.cp.map(at):r.cp,
             rail:r.rail?r.rail.map(([x,w,k])=>[at(x),at(x+w)-at(x),k]):r.rail});
        if(typeof extra==='string') extra=remapX(extra,LBL,at);
        if(typeof under==='string') under=remapX(under,LBL,at);
        opts={...opts, breaks:el.bands.map(b=>[b.p0,b.p1,''])};
      }
    }
  }
  if(under&&typeof under==='object'){ opts=under; under=''; }
  /**
   * Ribbons are DERIVED, not placed.
   *
   * One request can report several steps and they are all queued at the same
   * instant, so a ribbon is every row whose step was queued at the same moment,
   * drawn at that moment. Figures used to hand over an x and a list of row
   * indices, which is the same fact written a second time -- and it drifted:
   * ribbons hung off rows that had none and were missing between rows that
   * should have had one.
   *
   * The mark a ribbon threads loses its halo, because the ribbon is what the
   * circle is sitting on. That follows from the ribbon, so it is derived too.
   */
  const ribs=opts.noRib?[]:R.ribbonGroups(rows);
  if(ribs.length){
    under=ribs.map(g=>ribbon(g.x,g.rows.map(i=>cy(i)),
      opts.ribO!=null?{o:opts.ribO}:{})).join('')+under;
    rows=rows.map((r,i)=>{
      const on=ribs.find(g=>g.rows.includes(i));
      return on?{...r,noHalo:[...(r.noHalo||[]),on.x]}:r;
    });
  }
  const framed=opts.frame!==undefined?!!opts.frame:FRAMED;
  const ownRun=rows.some(r=>r.run||r.n==='Run');
  const above=framed?(ownRun?0:FRAME_ROWS_ABOVE):0;
  const n=opts.rowCount||rows.length;
  const lastY=rows.length?rowYs(rows)[rows.length-1]:cy(0)-ROW;
  // Only the Run row is extra: the finalization is one of `rows`.
  const h=lastY+ROW*(1+above)+TOP+(opts.pad||0)
    -(rows.length?0:ROW);
  /**
   * How far down the LOWEST ROW sits, in gaps of each pitch — which is what a
   * full-height mark should span.
   *
   * Not `h`: the figure's nominal height carries a row of slack below the last
   * row, and a band drawn to it held the box open by that much on exactly the
   * figure that had one. The compressed figures were the ones with the empty
   * strip underneath.
   */
  const _st=rowSteps(rows), _tail=_st.length?_st[_st.length-1]:[0,0];
  const lowRow=above+_tail[0], lowSpan=_tail[1];
  // Half a row of clearance past the last one, so the band still crosses a row
  // whose bar or mark hangs below the centre line — and half a ROW, not a fixed
  // number, so it keeps clearing it when the pitch slider moves.
  const bandRows=lowRow+0.5;
  const bandH=cy(0)+ROW*bandRows+SPAN_ROW*lowSpan+2;
  const figH=`calc(${bandH}px + (var(--geo-row,${ROW}px) - ${ROW}px) * ${bandRows}`+
    ` + (var(--geo-span,${SPAN_ROW}px) - ${SPAN_ROW}px) * ${lowSpan})`;
  const ends=rows.flatMap(r=>(r.segs||[]).map(([,x,w])=>x+w));
  const max=ends.length?Math.max(...ends):100;
  /**
   * How much the drawn content is scaled up to fill the width.
   *
   * A figure laid out from a duration is already normalised -- layout() put it
   * across the whole axis -- so there is nothing to fit and k is 1. Only a
   * figure whose positions were written by hand can come up short.
   *
   * And it is only ever a scale UP: stretch() is a no-op at or below 1, so a k
   * below 1 does not shrink anything, it just tells everything downstream a
   * scale that was never applied. That is what put the compression bands 14%
   * away from the bars they were cutting.
   */
  const k=opts.scale||(opts.ms!=null?1
    :(max>0?Math.max(1,Math.min(100/max, R.GEOM.MAX_SCALE)):1));
  // Rows are placed from their own pitches, so a block of span rows packs
  // tighter than the trace around it.
  const ys=rowYs(rows), st=rowSteps(rows);
  const body=under+rows.map((r,i)=>
    `<g class="r" style="--i:${st[i][0]};--s:${st[i][1]}">`+
      (r.group ? groupRowAt(i,{n:r.group.n, x:0, w:r.group.w, members:r.group.members,
                               note:r.group.note}) : row(i,r,k,ys[i]))+
    `</g>`).join('')+extra;
  const M=opts.margin||0;
  // Annotations are placed after the stretch, in final coordinates, so a
  // leader lands on the bar it points at rather than being scaled off it.
  const XS=p=>LBL+(px(p)-LBL)*k;
  const DY=ROW*above;
  const over=opts.over?opts.over(XS,i=>cy(i)+DY):'';
  /**
   * The cables into whichever row is in focus, derived.
   *
   * Every figure gets them, because "what caused this row" is a property of the
   * trace and not of the figure that happens to be drawing it. The focus is the
   * selected row, or one a figure names outright; without one there is nothing
   * to point at and no cables are drawn.
   */
  const focusRow = opts.focus!=null ? opts.focus : rows.findIndex(r=>r.sel);
  // `attributed:false` is a figure saying the trace never reported what caused
  // the row. Timing alone would answer it -- some earlier step always finished
  // just before -- and answering it from timing is the guess the design refuses
  // to make, so the figure has to be able to say the link is not there.
  const cab = (focusRow<0||focusRow==null||opts.attributed===false) ? ''
    : R.cables(rows,focusRow)
        .map(c=>wire(XS(c.x),ys[c.from]+DY,XS(c.q),ys[c.to]+DY,{i:Math.abs(c.to-c.from)-1}))
        .join('');
  /**
   * The outbound lineage marker, placed from the row that owns it.
   *
   * It used to be handed absolute coordinates by the figure -- and in the one
   * fixture that used it the figure indexed the rows it declared while the
   * drawing used the rows left after the Run and Finalization were dropped, so
   * the marker sat one row below the step whose events started the runs.
   *
   * Drawn in the over layer rather than in the row, because it is a fixed-size
   * glyph: inside the row it would be scaled horizontally with the time axis.
   */
  const lin=rows.map((r,i)=>{
    if(!r.lineage) return '';
    const segs=r.segs||[];
    if(!segs.length) return '';
    const end=Math.max(...segs.map(([,x,w])=>x+w));
    return `<g class="r" style="--i:${above+st[i][0]};--s:${st[i][1]}">`+
      lineage(XS(end), ys[i]+DY, r.lineage)+'</g>';
  }).join('');
  const inner=framed
    ? `<g class="dy" style="--a:${above}">${stretch(body,LBL,k)}</g>`
    : stretch(body,LBL,k);
  let ctx='';
  if(framed){
    const F=traceFrame(rows,k,{hasOwnRun:ownRun,breaks:opts.breaks||[],
      running:!!opts.running,lead:leadLabel,trimmed:R.leadingQueue(rows)<=0.01});
    /**
     * The frame is drawn like everything else.
     *
     * It used to be blurred and desaturated to say "context, not the subject",
     * which was one more thing to read on a page whose whole job is to be
     * readable -- and it fought every figure whose point WAS the Run row. The
     * frame is already quieter than the rows: it is thinner, and it is grey
     * wherever nothing was executing.
     */
    ctx=stretch(F.sharp,LBL,k)+stretch(F.soft,LBL,k);
  }
  // The compressed band blurs whatever runs through it. Done by drawing the
  // whole figure a second time, clipped to the band and filtered — SVG has no
  // backdrop-filter, and a flat scrim would hide the row rather than soften it.
  /**
   * The band is drawn OUTSIDE the group the rows are stretched in, so it has to
   * be given positions in the same space they end up in.
   *
   * A figure whose content collapses -- a two-second sleep compressed to a band
   * leaves 25 units of a 100-unit axis -- gets stretched to fill the width, by
   * 3x here. The band was placed at the unstretched position, so it marked
   * 105px while the Run row tore at 187px over the sleep it was supposed to be
   * cutting. Since px() is linear from LBL, the stretch is just a factor.
   */
  const cmp=compression((opts.breaks||[]).map(([a,b,t])=>[a*k,b*k,t]), bandH, ++UID);
  /**
   * Ticks go on the figures that are a run -- the same ones that carry the Run
   * row. A Concepts figure is about one mark or one bar and invents whatever
   * timings it needs, so putting a clock over it would read meaning into
   * numbers that have none.
   */
  const ax=(framed && axisAt)
    ? tickStrip(axisAt, axisTotal, opts.unit, k, (opts.breaks||[]).map(b=>[b[0]*k, b[1]*k]))
    : '';
  const AXH=ax?R.AXIS.strip:0;   // k is 1 unless the content was scaled up
  // Only the figure's own rows are blurred. The Run row is deliberately left
  // sharp: its torn track is what says "compressed here", and blurring the one
  // element carrying that message defeats drawing it at all.
  /**
   * No blur through the band.
   *
   * It was a second copy of the whole figure, clipped and filtered, to say
   * "this width is not to scale". Three things already say that and say it
   * without a duplicate: the scrim, the rules at both edges, and the Run row's
   * own track tearing. What the blur added was a halo over whatever it passed,
   * and a clip path per figure whose id collided with the next one's.
   */
  const blurred='';
  const nr=n+above;
  /**
   * Every figure carries the events it was drawn from.
   *
   * The build draws it once so the page paints without waiting for script, and
   * the component can redraw the same figure from the same events when
   * something changes what a drawing means -- a feature toggled off, say. Both
   * go through fig(); this is what lets the second one happen at all.
   */
  const fid=++FIG_ID;
  // Whether the figure was FRAMED is part of what it is, and it comes from a
  // module-level default as often as from the call -- so it is recorded as
  // resolved. Without it the component framed everything it redrew, putting a
  // Run row and a finalization under the Concepts figures, which are about one
  // mark or one bar and have no run to overview.
  FIGURES.push({id:fid, rows:rows0, extra:extra0, label, under:under0,
                opts:{...opts0, frame:framed}});
  return `<svg data-fig="${fid}" viewBox="${-M} ${-AXH} ${W+M*2} ${h+AXH}" style="--nr:${nr};--fig-h0:${bandH}px;--fig-h:${figH}" role="img" aria-label="${label}">`+
    HATCH+BLURDEF+cmp.clip+gutter(bandH)+ctx+inner+blurred+`<g class="nofit">${cmp.over}</g>`+ax+over+cab+lin+`</svg>`;
}

const BLURDEF='';

/**
 * The tie between one discovery request and every step it queued.
 *
 * A tapered ribbon threaded down the members' queued circles: it swells to just
 * wider than a circle where one sits, and pinches between them. Drawn under the
 * rows, so each circle reads as sitting ON it rather than beside it — the
 * grouping costs no row and favours no member.
 */
export function ribbon(xp, ys, {o=1, w=3.4, gaps}={}){
  if(ys.length<2) return '';
  const x=px(xp);
  const span=ys[ys.length-1]-ys[0];
  // `gaps` is how many row pitches it spans, so CSS can restretch it when the
  // pitch moves. Without it the ribbon kept its generated height while the rows
  // it threads slid out from under it.
  const n=gaps!=null?gaps:Math.max(1,Math.round(span/ROW));
  const i0=((ys[0]-(TOP+9))/ROW).toFixed(4);
  return `<rect class="rib" style="--n:${n};--i:${i0}" x="${x-w/2}" y="${ys[0]}" width="${w}" `+
    `height="${span}" fill="${C.disc}" opacity="${0.95*o}"/>`;
}
