import { GEOM } from './vocabulary.mjs';
export const {W,LBL,RGT,PLOT,ROW,TOP}=GEOM;
export { GEOM };
export { C, EV, HATCH, dot, base, isFail, FILL, COMPUTE, NOCOMPUTE, ACTIVE, OPEN, paint, autoDots, discovered, lineage, barSvg, markSvg, cyOf, pxOf, BAR_INFO, EVENT_INFO, EVC, SUBSTANCE, substanceCSS, eventCSS } from './vocabulary.mjs';
import { C, EV, HATCH, dot, base, paint, autoDots, OPEN, barSvg, causalRibbon as _cr, wire, fillGaps, stretch, BAR_INFO, EVENT_INFO } from './vocabulary.mjs';
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
  const {n,segs:rawSegs=[],dim=1,dots,note,sel,noHalo}=r;
  const segs=fillGaps(rawSegs.map(([k,x,w])=>({kind:k,x,w}))).map(g=>[g.kind,g.x,g.w]);
  const y=cy(i); let s='';
  if(sel) s+=`<rect x="${LBL-3}" y="${y-7.5}" width="${PLOT+6}" height="15" fill="${C.acc}" opacity=".13" rx="2"/>`;
  s+=`<text x="2" y="${y+2.5}" ${MONO} font-size="7" fill="${C.mut}" opacity="${dim}">${n}</text>`;
  segs.forEach(([k,a,w])=>{ s+=barSvg(k,a,w,y,{k:1,floor:sc,o:dim}); });
  const auto=autoDots(segs.map(([k,a,w])=>({kind:k,x:a,w})));
  (dots||auto).forEach(d=>{const onRib=(noHalo||[]).some(p=>Math.abs(p-d.p)<0.01); s+=dot(px(d.p),y,onRib?'ribbon':(d.c||C.mut),dim,3,!onRib);});
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
export function axis(y, ticks, {tier2=[], brk=[]}={}){
  let s=`<line x1="${LBL}" y1="${y}" x2="${LBL+PLOT}" y2="${y}" stroke="${C.idle}" stroke-width="1"/>`;
  for(const b of brk)
    s+=`<rect x="${px(b[0])}" y="${y-3}" width="${((b[1]-b[0])/100)*PLOT}" height="6" fill="url(#hx-idle)"/>`+
       `<line x1="${px(b[0])}" y1="${y-4}" x2="${px(b[0])}" y2="${y+4}" stroke="${C.idle}" stroke-width="1" stroke-dasharray="1.5 1.5"/>`+
       `<line x1="${px(b[1])}" y1="${y-4}" x2="${px(b[1])}" y2="${y+4}" stroke="${C.idle}" stroke-width="1" stroke-dasharray="1.5 1.5"/>`;
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

export function fig(rows,extra='',label='',under='',opts={}){
  if(under&&typeof under==='object'){ opts=under; under=''; }
  if(opts.rib){
    under=ribbon(opts.rib.x,opts.rib.rows.map(i=>cy(i)))+under;
    rows=rows.map((r,i)=>opts.rib.rows.includes(i)?{...r,noHalo:[opts.rib.x]}:r);
  }
  const h=TOP*2+ROW*(opts.rowCount||rows.length);
  const ends=rows.flatMap(r=>(r.segs||[]).map(([,x,w])=>x+w));
  const max=ends.length?Math.max(...ends):100;
  const k=opts.scale||(max>0?Math.min(86/max,3):1);
  const body=under+rows.map((r,i)=>row(i,r,k)).join('')+extra;
  const M=opts.margin||0;
  // Annotations are placed after the stretch, in final coordinates, so a
  // leader lands on the bar it points at rather than being scaled off it.
  const XS=p=>LBL+(px(p)-LBL)*k;
  const over=opts.over?opts.over(XS,cy):'';
  return `<svg viewBox="${-M} 0 ${W+M*2} ${h+(opts.pad||0)}" role="img" aria-label="${label}">`+
    HATCH+stretch(body,LBL,k)+over+`</svg>`;
}

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
