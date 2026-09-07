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
  {n:'a',at:[['queued',0],['started',6],['ok',36]]},
  {n:'b',at:[['queued',0],['started',6],['ok',48]]},
  {n:'c',at:[['queued',0],['started',6],['ok',42]]},
  {n:'d',at:[['started',56],['ok',68],['queued',68],['started',72],['ok',94]],reported:1},
];
O.coalCurve=fig(ROWS_COAL,
  arrow(px(36),cy(0),px(56),cy(3))+arrow(px(48),cy(1),px(56),cy(3))+arrow(px(42),cy(2),px(56),cy(3)),
  'coalesce with curves');
O.coalRibbon=fig(ROWS_COAL,
  ribbonTo([{x:36,row:0},{x:48,row:1},{x:42,row:2}],{x:56,row:3}),
  'coalesce with an orthogonal ribbon');

const ROWS_SEQ=[
  {n:'a',at:[['queued',0],['started',8],['ok',38]]},
  {n:'b',at:[['queued',44],['started',50],['ok',84]]},
];
O.seqCurve=fig(ROWS_SEQ, arrow(px(38),cy(0),px(50),cy(1)), 'sequential with a curve');
O.seqRibbon=fig(ROWS_SEQ, ribbonTo([{x:38,row:0}],{x:50,row:1}), 'sequential with a ribbon');

// two branches at once — where curves stop working
const ROWS_CH=[
  {n:'left-1',at:[['queued',0],['started',6],['ok',30]]},
  {n:'right-1',at:[['queued',0],['started',6],['ok',38]]},
  {n:'left-2',at:[['started',36],['ok',44],['queued',44],['started',47],['ok',67]],reported:1},
  {n:'right-2',at:[['started',42],['ok',50],['queued',50],['started',53],['ok',77]],reported:1},
];
O.chCurve=fig(ROWS_CH,
  arrow(px(30),cy(0),px(36),cy(2))+arrow(px(38),cy(1),px(42),cy(3)),
  'two branches with curves');
O.chRibbon=fig(ROWS_CH,
  ribbonTo([{x:30,row:0}],{x:36,row:2})+ribbonTo([{x:38,row:1}],{x:42,row:3}),
  'two branches with ribbons');

// a race: the winner is a dependency, the losers are not
const ROWS_RACE=[
  {n:'fast',at:[['queued',0],['started',6],['ok',28]]},
  {n:'slow',at:[['queued',0],['started',6],['ok',52]]},
  {n:'next',at:[['started',34],['ok',44],['queued',44],['started',47],['ok',73]],reported:1},
];
O.race=fig(ROWS_RACE.map((r,i)=>i===1?{...r,dim:.28}:r),
  ribbonTo([{x:28,row:0}],{x:34,row:2}),
  'a race: the winner has a line, the loser has none');

fs.writeFileSync(HERE+'causal.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
