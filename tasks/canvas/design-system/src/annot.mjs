import {C} from './vocabulary.mjs';
const PEN='var(--pen)';
const HAND="font-family='Caveat, Segoe Script, cursive'";

/** A wobble so a line reads as drawn rather than plotted. */
const jit=(n,s)=>n+(Math.sin(s*12.9898)*43758.5453%1)*1.6-0.8;

/**
 * A hand-drawn callout: a label with a curved leader pointing at a target.
 * `from` is where the text sits, `to` is what it refers to.
 */
export function callout(fx,fy,tx,ty,text,{size=11,anchor='start'}={}){
  const mx=(fx+tx)/2, my=(fy+ty)/2;
  const c1x=jit(mx,fx), c1y=jit(fy,tx);
  const d=`M${fx.toFixed(1)} ${fy.toFixed(1)} Q${c1x.toFixed(1)} ${c1y.toFixed(1)} ${tx.toFixed(1)} ${ty.toFixed(1)}`;
  const a=Math.atan2(ty-c1y,tx-c1x);
  const head=(o)=>`<path d="M${(tx-9*Math.cos(a-o)).toFixed(1)} ${(ty-9*Math.sin(a-o)).toFixed(1)} L${tx.toFixed(1)} ${ty.toFixed(1)}" stroke="${PEN}" stroke-width="1.3" fill="none" stroke-linecap="round"/>`;
  return `<g filter="url(#hx-wire)"><path d="${d}" stroke="${PEN}" stroke-width="1.3" fill="none" stroke-linecap="round" opacity=".95"/>`+
    head(0.45)+head(-0.45)+`</g>`+
    `<text x="${fx.toFixed(1)}" y="${fy.toFixed(1)}" ${HAND} font-size="${size}" fill="${PEN}" text-anchor="${anchor}" dy="-4">${text}</text>`;
}

/** A bracket under a span, with a label. */
export function underline(x1,x2,y,text,{size=11}={}){
  return `<path d="M${x1} ${y} L${x1} ${y+4} L${x2} ${y+4} L${x2} ${y}" stroke="${PEN}" stroke-width="1.2" fill="none" stroke-linecap="round" opacity=".85"/>`+
    `<text x="${((x1+x2)/2).toFixed(1)}" y="${y+16}" ${HAND} font-size="${size}" fill="${PEN}" text-anchor="middle">${text}</text>`;
}
