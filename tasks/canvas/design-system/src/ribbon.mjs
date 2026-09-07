const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,ribbon,arrow,tag,C,W,LBL,PLOT} from './micro.mjs';
const O={};

// Fan-out: the request stays in a's row, the ribbon ties all three queues.
const Q=24;
O.fan=fig([
  {n:'req + a',at:[['started',8],['ok',24],['queued',24],['started',30],['ok',64]],reported:1,note:'34ms',noHalo:[Q]},
  {n:'b',at:[['queued',24],['started',30],['ok',72]],note:'42ms',noHalo:[Q]},
  {n:'c',at:[['queued',24],['started',30],['ok',58]],note:'28ms',noHalo:[Q]},
],'','fan-out tied by a ribbon',ribbon(Q,[cy(0),cy(1),cy(2)]));

O.compare=fig([
  {n:'req + a',at:[['started',8],['ok',24],['queued',24],['started',30],['ok',64]],reported:1},
  {n:'b',at:[['queued',24],['started',30],['ok',72]]},
  {n:'c',at:[['queued',24],['started',30],['ok',58]]},
],'','fan-out without the ribbon');

// Wide: twelve members, one ribbon.
const many=[...Array(8)].map((_,i)=>({n:'w'+i,segs:[['idle',Q,5],['good',29,20+i*4]],note:(20+i*4)+'ms',noHalo:[Q]}));
O.wide=fig([
  {n:'req + w0',at:[['started',10],['ok',24],['queued',24],['started',29],['ok',45]],reported:1,note:'16ms',noHalo:[Q]},
  ...many,
],'','wide fan-out',ribbon(Q,[...Array(9)].map((_,i)=>cy(i))));

// It survives a ragged queue: members that queue at slightly different times.
O.ragged=fig([
  {n:'req + a',at:[['started',8],['ok',24],['queued',24],['started',32],['ok',62]],reported:1,note:'30ms'},
  {n:'b',at:[['queued',26],['started',34],['ok',70]],note:'36ms'},
  {n:'c',at:[['queued',25],['started',33],['ok',57]],note:'24ms'},
],'','ragged queue times',ribbon(25,[cy(0),cy(1),cy(2)]));

fs.writeFileSync(HERE+'ribbon.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
