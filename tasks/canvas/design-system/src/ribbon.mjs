const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,ribbon,arrow,tag,C,W,LBL,PLOT} from './micro.mjs';
const O={};

// Fan-out: the request stays in a's row, the ribbon ties all three queues.
const Q=24;
O.fan=fig([
  {n:'req + a',segs:[['disc',8,16],['idle',Q,6],['good',30,34]],note:'34ms',noHalo:[Q]},
  {n:'b',segs:[['idle',Q,6],['good',30,42]],note:'42ms',noHalo:[Q]},
  {n:'c',segs:[['idle',Q,6],['good',30,28]],note:'28ms',noHalo:[Q]},
],'','fan-out tied by a ribbon',ribbon(Q,[cy(0),cy(1),cy(2)]));

O.compare=fig([
  {n:'req + a',segs:[['disc',8,16],['idle',Q,6],['good',30,34]]},
  {n:'b',segs:[['idle',Q,6],['good',30,42]]},
  {n:'c',segs:[['idle',Q,6],['good',30,28]]},
],'','fan-out without the ribbon');

// Wide: twelve members, one ribbon.
const many=[...Array(8)].map((_,i)=>({n:'w'+i,segs:[['idle',Q,5],['good',29,20+i*4]],note:(20+i*4)+'ms',noHalo:[Q]}));
O.wide=fig([
  {n:'req + w0',segs:[['disc',10,14],['idle',Q,5],['good',29,16]],note:'16ms',noHalo:[Q]},
  ...many,
],'','wide fan-out',ribbon(Q,[...Array(9)].map((_,i)=>cy(i))));

// It survives a ragged queue: members that queue at slightly different times.
O.ragged=fig([
  {n:'req + a',segs:[['disc',8,16],['idle',24,8],['good',32,30]],note:'30ms'},
  {n:'b',segs:[['idle',26,8],['good',34,36]],note:'36ms'},
  {n:'c',segs:[['idle',25,8],['good',33,24]],note:'24ms'},
],'','ragged queue times',ribbon(25,[cy(0),cy(1),cy(2)]));

fs.writeFileSync(HERE+'ribbon.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
