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
export function runProfile(i,{to=86,intervals=[],resolved,n='Run'},sc=1){
  const y=cy(i);
  const col=v=> v.ok===false?C.bad : v.ok==='stop'?C.mut : v.ok==='mix'?C.mix : C.good;
  let s=`<text x="2" y="${y+2.5}" ${MONO} font-size="7" fill="${C.mut}">${n}</text>`;
  s+=`<rect x="${LBL}" y="${y-2.5}" width="${((to/100)*PLOT).toFixed(1)}" height="5" rx="1.2" fill="${C.idle}"/>`;
  for(const v of intervals){
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

export function row(i,r,sc=1){
  if(r.run) return runProfile(i,r,sc);
  const {n,segs:rawSegs=[],dim=1,dots,note,sel,noHalo,lit,litDots}=r;
  const segs=fillGaps(rawSegs.map(([k,x,w])=>({kind:k,x,w}))).map(g=>[g.kind,g.x,g.w]);
  const y=cy(i); let s='';
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
  s+=`<text x="2" y="${y+2.5}" ${MONO} font-size="7" fill="${C.mut}" opacity="${dim}">${n}</text>`;
  segs.forEach(([k,a,w])=>{ s+=barSvg(k,a,w,y,{k:1,floor:sc,o:litBar(k,a)}); });
  const auto=autoDots(segs.map(([k,a,w])=>({kind:k,x:a,w})));
  (dots||auto).forEach(d=>{const onRib=(noHalo||[]).some(p=>Math.abs(p-d.p)<0.01); s+=dot(px(d.p),y,onRib?'ribbon':(d.c||C.mut),litDot(d.p),3,!onRib);});
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
    if(parts.length) s+=`<rect class="rowhit" x="${LBL-6}" y="${y-8}" width="${PLOT+12}" height="16" fill="transparent" data-row="${n}" data-parts='${JSON.stringify(parts).replace(/'/g,"&apos;")}'/>`;
  }
  if(note && NOTES){
    const last=segs[segs.length-1];
    const at=last?px(last[1]+last[2])+9:LBL;
    s+=`<text x="${at.toFixed(1)}" y="${y+2.5}" ${MONO} font-size="6.5" fill="${C.mut}" opacity="${dim}">${note}</text>`;
  }
  return s;
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
      s+=`<line x1="${x}" y1="${top?top:y-4}" x2="${x}" y2="${y+4}" stroke="${C.idle}" stroke-width="1" stroke-dasharray="1.5 2" opacity=".55"/>`;
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
 * Two frame rows sit above the figure (the minimap strip, then Run) and one
 * below (Finalization), so the figure's own rows start at slot 2.
 */
export const FRAME_ROWS_ABOVE=2;
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

export function traceFrame(rows,k,{end,hasOwnRun,pad=0}){
  const y0=TOP+4;                                  // minimap strip
  // A figure with its own axis draws it below its last row, so finalization
  // has to clear the axis rather than land on the ticks.
  const finY=cy(FRAME_ROWS_ABOVE+rows.length)+pad;
  // The minimap is the same trace in the same order — one hairline per row,
  // at the position that row's work occupied. Never a different set of steps.
  // One hairline per ROW, not per segment. Merging them into a band made the
  // minimap a second Run row — same position, same colour, same weight — and
  // two drawings of one fact is the defect this whole language exists to avoid.
  // It is granular where the Run row is a profile: that is the difference.
  let mini=`<rect x="${LBL}" y="${y0+2.6}" width="${PLOT}" height="1" fill="${C.idle}"/>`;
  rows.forEach((r,ri)=>{
    const cs=(r.segs||[]).filter(([kd])=>COMPUTE.has(base(kd)));
    if(!cs.length) return;
    const a=Math.min(...cs.map(([,x])=>x)), b=Math.max(...cs.map(([,x,w])=>x+w));
    const bad=cs.some(([kd])=>isFail(kd));
    mini+=`<rect x="${px(a).toFixed(1)}" y="${(y0+0.6+ri*1.5).toFixed(1)}" width="${Math.max(0.8,((b-a)/100)*PLOT).toFixed(1)}" height="1.1" rx="0.4" fill="${bad?C.bad:C.good}" opacity=".95"/>`;
  });
  mini+=tag(2,y0+5,'minimap');
  // The Run row is a profile of where the elapsed time went, not a restatement
  // of the status: grey ground, coloured only where the SDK was executing.
  const intervals=[];
  rows.forEach(r=>(r.segs||[]).forEach(([kd,x,w])=>{
    if(COMPUTE.has(base(kd))) intervals.push({a:x,b:x+w,ok:!isFail(kd)});
  }));
  // A figure that already draws its own Run row keeps it; two Run rows would
  // be the same interval drawn twice, which is the defect this language exists
  // to prevent.
  const run=hasOwnRun?'':runProfile(1,{to:Math.min(end+6,96),intervals,resolved:EV.ok},k);
  // Finalization is platform work: its own row, recessive label, still coloured.
  // Finalization is a discovery request like any other — it asks the SDK what
  // is next and the answer is "nothing". So it is queued, it waits, it starts,
  // and the bar is `disc`: **your app executes for it**, and you are billed for
  // it. Drawing it as a bare green bar hid both the wait and the compute.
  const fx=Math.min(end+2,90), fq=2.2, fw=4.5;
  const fin=tag(2,finY+2.5,'Finalization')+
    barSvg('idle',fx,fq,finY,{k:1,floor:k,o:1})+
    barSvg('disc',fx+fq,fw,finY,{k:1,floor:k,o:1})+
    dot(px(fx),finY,EV.queued)+
    dot(px(fx+fq),finY,EV.started)+
    dot(px(fx+fq+fw),finY,EV.ok);
  // The frame shares the row grid with the figure, and the inner content keeps
  // its own `cy` attributes because it is translated as a group — so the Run
  // row and the figure's second row read as one row to anything parsing the
  // SVG back. Tag the frame's marks so the validator skips them: they are
  // context, not rows under test.
  return (mini+run+fin).replace(/<circle class="ev /g,'<circle class="ev ctx ');
}

export function fig(rows,extra='',label='',under='',opts={}){
  if(under&&typeof under==='object'){ opts=under; under=''; }
  if(opts.rib){
    under=ribbon(opts.rib.x,opts.rib.rows.map(i=>cy(i)))+under;
    rows=rows.map((r,i)=>opts.rib.rows.includes(i)?{...r,noHalo:[opts.rib.x]}:r);
  }
  const framed=opts.frame!==undefined?!!opts.frame:FRAMED;
  const n=opts.rowCount||rows.length;
  const h=TOP*2+ROW*(n+(framed?FRAME_ROWS_ABOVE+1:0))+(framed?(opts.pad||0):0);
  const ends=rows.flatMap(r=>(r.segs||[]).map(([,x,w])=>x+w));
  const max=ends.length?Math.max(...ends):100;
  const k=opts.scale||(max>0?Math.min(86/max,3):1);
  const body=under+rows.map((r,i)=>row(i,r,k)).join('')+extra;
  const M=opts.margin||0;
  // Annotations are placed after the stretch, in final coordinates, so a
  // leader lands on the bar it points at rather than being scaled off it.
  const XS=p=>LBL+(px(p)-LBL)*k;
  const DY=framed?ROW*FRAME_ROWS_ABOVE:0;
  const over=opts.over?opts.over(XS,i=>cy(i)+DY):'';
  const inner=framed
    ? `<g transform="translate(0,${DY})">${stretch(body,LBL,k)}</g>`
    : stretch(body,LBL,k);
  const ctx=framed
    ? `<g filter="url(#ctxblur)" opacity="${CTX_O}">${stretch(traceFrame(rows,k,{end:max,hasOwnRun:rows.some(r=>r.run),pad:opts.pad||0}),LBL,k)}</g>`
    : '';
  return `<svg viewBox="${-M} 0 ${W+M*2} ${h+(opts.pad||0)}" role="img" aria-label="${label}">`+
    HATCH+BLURDEF+ctx+inner+over+`</svg>`;
}

const BLURDEF=`<defs><filter id="ctxblur" x="-4%" y="-30%" width="108%" height="160%">`+
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
