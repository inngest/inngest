export const W=680, LBL=92, RGT=118, PLOT=W-LBL-RGT, ROW=24, TOP=8;
export { C, EV, HATCH, dot, base, isFail, FILL, COMPUTE, NOCOMPUTE, ACTIVE, OPEN, paint, autoDots, discovered, lineage } from './vocabulary.mjs';
import { C, EV, HATCH, dot, base, paint, autoDots, OPEN, causalRibbon as _cr, wire, fillGaps } from './vocabulary.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";
export const px=p=>LBL+(p/100)*PLOT;
export const cy=i=>TOP+12+ROW*i;
export { wire };
export const causal=(sources,target,opts={})=>_cr(sources,target,{px,cy,...opts});
export const head=(x,y,c=C.acc)=>`<path d="M${x-3.5} ${y-2.6} L${x} ${y} L${x-3.5} ${y+2.6} Z" fill="${c}"/>`;
export const arrow=(x1,y1,x2,y2)=>{
  const d=Math.max(14,Math.abs(x2-x1)*0.55);
  return `<path d="M${x1} ${y1} C ${x1+d} ${y1}, ${x2-d} ${y2}, ${x2-4} ${y2}" fill="none" stroke="${C.acc}" stroke-width="1.4"/>`+head(x2,y2);
};

/**
 * One row. `segs` are {x,w,kind} in plot percent; `dots` are {p,c} in percent.
 * Bars are derived from the segments; circles are placed at transitions.
 */
export function row(i,{name,note,dur,segs:rawSegs,dots=[],dim=1,sel=false,noHalo}){
  const segs=fillGaps(rawSegs||[]);
  const y=cy(i);
  let s='';
  if(sel) s+=`<rect x="${LBL-4}" y="${y-10}" width="${PLOT+8}" height="20" fill="${C.acc}" opacity=".07" rx="2"/>`;
  s+=`<text x="2" y="${y+3}" ${MONO} font-size="8.5" fill="${C.mut}" opacity="${dim}">${name}</text>`;
  segs.forEach(g=>{
    const h=7.5;
    s+=`<rect x="${px(g.x).toFixed(1)}" y="${y-h/2}" width="${Math.max(1.4,(g.w/100)*PLOT).toFixed(1)}" height="${h}" rx="1.2" fill="${paint(g.kind)}" opacity="${dim}"/>`;
  });
  const auto = dots.length ? [] : autoDots(segs);
  (dots.length?dots:auto).forEach(d=>s+=dot(px(d.p),y,d.c===undefined?C.neut:d.c,dim,3.6,!(noHalo||[]).some(p=>Math.abs(p-d.p)<0.01)));
  const right=[note,dur].filter(Boolean).join('  ');
  if(right) s+=`<text x="${W-4}" y="${y+3}" ${MONO} font-size="7.5" fill="${C.mut}" text-anchor="end" opacity="${dim}">${right}</text>`;
  return s;
}

export function panel(rows,extra='',label=''){
  const h=TOP*2+ROW*rows.length;
  return `<svg viewBox="0 0 ${W} ${h}" role="img" aria-label="${label}">`+HATCH+
    rows.map((r,i)=>r.run?runRow(i,r.run,r.lbl):row(i,r)).join('')+extra+`</svg>`;
}

/**
 * The Run row: a compute profile. Grey is elapsed time we were not on the
 * user's app; colour is time we were. A slice takes green or red only if every
 * step running in it agrees — mixed slices fall back to the discovery blue,
 * which is unambiguous here only because the Run row never draws discovery.
 */
export function runRow(i, intervals, label){
  const y=cy(i);
  const edges=[...new Set(intervals.flatMap(v=>[v.a,v.b]))].sort((p,q)=>p-q);
  let s=`<text x="2" y="${y+3}" font-family="JetBrains Mono, monospace" font-size="8.5" fill="${C.mut}">Run</text>`;
  s+=`<rect x="${LBL}" y="${y-2.5}" width="${PLOT}" height="5" rx="1.2" fill="${C.idle}"/>`;
  for(let k=0;k<edges.length-1;k++){
    const a=edges[k], b=edges[k+1];
    const live=intervals.filter(v=>v.a<=a && v.b>=b);
    if(!live.length) continue;
    const allOk=live.every(v=>v.ok), allBad=live.every(v=>!v.ok);
    const fill=allOk?C.good:allBad?C.bad:C.mix;
    // Succeeded work may shrink to nothing; failure stays findable.
    const w=Math.max(allBad?3.2:1.4, ((b-a)/100)*PLOT);
    s+=`<rect x="${px(a).toFixed(1)}" y="${y-4}" width="${w.toFixed(1)}" height="8" rx="1.2" fill="${fill}"/>`;
  }
  s+=dot(LBL,y,EV.queued)+dot(LBL+PLOT,y,EV.ok);
  if(label) s+=`<text x="${W-4}" y="${y+3}" font-family="JetBrains Mono, monospace" font-size="7.5" fill="${C.mut}" text-anchor="end">${label}</text>`;
  return s;
}
