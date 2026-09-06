const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,arrow,tag,C,W,LBL,PLOT,ROW} from './micro.mjs';
const O={};
const B=C.disc, WGT=2.2;

/** Orthogonal causal run: out from each source, merge on one spine, in to the target. */
function ribbonTo(sources, target, {o=1}={}){
  const xs=sources.map(s=>px(s.x)), ys=sources.map(s=>cy(s.row));
  const xJ=px(target.x)-14, yT=cy(target.row), xT=px(target.x);
  let d='';
  // each contributor runs right along its own row to the spine
  sources.forEach((s,i)=>{ d+=`<path d="M${xs[i]} ${ys[i]} H${xJ}" fill="none" stroke="${B}" stroke-width="${WGT}" opacity="${o}" stroke-linecap="round"/>`; });
  // one spine down to the target row
  const y0=Math.min(...ys, yT), y1=Math.max(...ys, yT);
  d+=`<path d="M${xJ} ${y0} V${y1}" fill="none" stroke="${B}" stroke-width="${WGT}" opacity="${o}" stroke-linecap="round"/>`;
  // and in to where the step actually starts
  d+=`<path d="M${xJ} ${yT} H${xT-4}" fill="none" stroke="${B}" stroke-width="${WGT}" opacity="${o}"/>`;
  d+=`<path d="M${xT-4} ${yT-3} L${xT} ${yT} L${xT-4} ${yT+3} Z" fill="${B}" opacity="${o}"/>`;
  return d;
}

const ROWS_COAL=[
  {n:'a',segs:[['idle',0,6],['good',6,30]]},
  {n:'b',segs:[['idle',0,6],['good',6,42]]},
  {n:'c',segs:[['idle',0,6],['good',6,36]]},
  {n:'d',segs:[['disc',56,12],['idle',68,4],['good',72,22]]},
];
O.coalCurve=fig(ROWS_COAL,
  arrow(px(36),cy(0),px(56),cy(3))+arrow(px(48),cy(1),px(56),cy(3))+arrow(px(42),cy(2),px(56),cy(3)),
  'coalesce with curves');
O.coalRibbon=fig(ROWS_COAL,
  ribbonTo([{x:36,row:0},{x:48,row:1},{x:42,row:2}],{x:56,row:3}),
  'coalesce with an orthogonal ribbon');

const ROWS_SEQ=[
  {n:'a',segs:[['idle',0,8],['good*',8,30]]},
  {n:'b',segs:[['idle',44,6],['good*',50,34]]},
];
O.seqCurve=fig(ROWS_SEQ, arrow(px(38),cy(0),px(50),cy(1)), 'sequential with a curve');
O.seqRibbon=fig(ROWS_SEQ, ribbonTo([{x:38,row:0}],{x:50,row:1}), 'sequential with a ribbon');

// two branches at once — where curves stop working
const ROWS_CH=[
  {n:'left-1',segs:[['idle',0,6],['good',6,24]]},
  {n:'right-1',segs:[['idle',0,6],['good',6,32]]},
  {n:'left-2',segs:[['disc',36,8],['idle',44,3],['good',47,20]]},
  {n:'right-2',segs:[['disc',42,8],['idle',50,3],['good',53,24]]},
];
O.chCurve=fig(ROWS_CH,
  arrow(px(30),cy(0),px(36),cy(2))+arrow(px(38),cy(1),px(42),cy(3)),
  'two branches with curves');
O.chRibbon=fig(ROWS_CH,
  ribbonTo([{x:30,row:0}],{x:36,row:2})+ribbonTo([{x:38,row:1}],{x:42,row:3}),
  'two branches with ribbons');

// a race: the winner is a dependency, the losers are not
const ROWS_RACE=[
  {n:'fast',segs:[['idle',0,6],['good',6,22]]},
  {n:'slow',segs:[['idle',0,6],['good',6,46]]},
  {n:'next',segs:[['disc',34,10],['idle',44,3],['good',47,26]]},
];
O.race=fig(ROWS_RACE.map((r,i)=>i===1?{...r,dim:.28}:r),
  ribbonTo([{x:28,row:0}],{x:34,row:2}),
  'a race: the winner has a line, the loser has none');

fs.writeFileSync(HERE+'causal.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
