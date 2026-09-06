const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,arrow,tag,ribbon,C,W,LBL,PLOT} from './micro.mjs';
const O={};
const Q=24;

// Sequential: the request becomes the step. Nothing to group.
O.seq=fig([
  {n:'a',segs:[['good',0,20]]},
  {n:'b',segs:[['idle',30,8],['good',38,34]],note:'discovered'},
],'','sequential');

// Fan-out without the tie: which steps did that request produce?
O.before=fig([
  {n:'req + a',segs:[['disc',8,16],['idle',Q,6],['good',30,34]],note:'34ms'},
  {n:'b',segs:[['idle',Q,6],['good',30,42]],note:'42ms'},
  {n:'c',segs:[['idle',Q,6],['good',30,28]],note:'28ms'},
  {n:'unrelated',segs:[['idle',Q,6],['good',30,50]],note:'50ms'},
],'','fan-out without the ribbon');

// With it: the ribbon covers exactly the members.
O.after=fig([
  {n:'req + a',segs:[['disc',8,16],['idle',Q,6],['good',30,34]],note:'34ms',noHalo:[Q]},
  {n:'b',segs:[['idle',Q,6],['good',30,42]],note:'42ms',noHalo:[Q]},
  {n:'c',segs:[['idle',Q,6],['good',30,28]],note:'28ms',noHalo:[Q]},
  {n:'unrelated',segs:[['idle',Q,6],['good',30,50]],note:'50ms'},
],'','fan-out with the ribbon',ribbon(Q,[cy(0),cy(1),cy(2)]));

// Twelve members, one ribbon, no extra rows.
const many=[...Array(8)].map((_,i)=>({n:'w'+(i+1),segs:[['idle',Q,5],['good',29,20+i*4]],note:(20+i*4)+'ms',noHalo:[Q]}));
O.wide=fig([
  {n:'req + w0',segs:[['disc',10,14],['idle',Q,5],['good',29,16]],note:'16ms',noHalo:[Q]},
  ...many,
],'','wide fan-out',ribbon(Q,[...Array(9)].map((_,i)=>cy(i))));

// Members that queue at slightly different moments.
O.ragged=fig([
  {n:'req + a',segs:[['disc',8,16],['idle',24,8],['good',32,30]],note:'30ms',noHalo:[24]},
  {n:'b',segs:[['idle',26,8],['good',34,36]],note:'36ms',noHalo:[26]},
  {n:'c',segs:[['idle',25,8],['good',33,24]],note:'24ms',noHalo:[25]},
],'','ragged queue times',ribbon(25,[cy(0),cy(1),cy(2)]));

// Coalesce keeps the arrows; there is nothing to group on the way in.
O.coalesce=fig([
  {n:'a',segs:[['good',2,26]]},
  {n:'b',segs:[['good',2,34]]},
  {n:'c',segs:[['good',2,30]]},
  {n:'d',segs:[['idle',40,16.0],['good',56,22]],note:'22ms'},
],'','coalesce');

fs.writeFileSync(HERE+'attribution.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
