const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,arrow,tag,ribbon,C,W,LBL,PLOT} from './micro.mjs';
const O={};
const Q=24;

// Sequential: the request becomes the step. Nothing to group.
O.seq=fig([
  {n:'a',at:[['started',0],['ok',20]]},
  {n:'b',at:[['queued',30],['started',38],['ok',72]],note:'discovered'},
],'','sequential');

// Fan-out without the tie: which steps did that request produce?
O.before=fig([
  {n:'req + a',at:[['queued',8],['started',13.6,'disc'],['ok',24],['planned',24],['started',30],['ok',64]],reported:1,note:'34ms'},
  {n:'b',at:[['planned',24],['started',30],['ok',72]],note:'42ms'},
  {n:'c',at:[['planned',24],['started',30],['ok',58]],note:'28ms'},
  {n:'unrelated',at:[['planned',24],['started',30],['ok',80]],note:'50ms'},
],'','fan-out without the ribbon');

// With it: the ribbon covers exactly the members.
O.after=fig([
  {n:'req + a',at:[['queued',8],['started',13.6,'disc'],['ok',24],['planned',24],['started',30],['ok',64]],reported:1,note:'34ms'},
  {n:'b',at:[['planned',24],['started',30],['ok',72]],note:'42ms'},
  {n:'c',at:[['planned',24],['started',30],['ok',58]],note:'28ms'},
  {n:'unrelated',at:[['planned',24],['started',30],['ok',80]],note:'50ms'},
],'','fan-out with the ribbon');

// Twelve members, one ribbon, no extra rows.
const many=[...Array(8)].map((_,i)=>({n:'w'+(i+1),segs:[['idle',Q,5],['good',29,20+i*4]],note:(20+i*4)+'ms'}));
O.wide=fig([
  {n:'req + w0',at:[['queued',10],['started',14.9,'disc'],['ok',24],['queued',24],['started',29],['ok',45]],reported:1,note:'16ms'},
  ...many,
],'','wide fan-out');

// Members that queue at slightly different moments.
O.ragged=fig([
  {n:'req + a',at:[['queued',8],['started',13.6,'disc'],['ok',24],['queued',24],['started',32],['ok',62]],reported:1,reports:['b','c'],note:'30ms'},
  {n:'b',at:[['queued',26],['started',34],['ok',70]],note:'36ms'},
  {n:'c',at:[['queued',25],['started',33],['ok',57]],note:'24ms'},
],'','ragged queue times');

// Coalesce keeps the arrows; there is nothing to group on the way in.
O.coalesce=fig([
  {n:'a',at:[['started',2],['ok',28]]},
  {n:'b',at:[['started',2],['ok',36]]},
  {n:'c',at:[['started',2],['ok',32]]},
  {n:'d',at:[['queued',40],['started',56],['ok',78]],note:'22ms'},
],'','coalesce');

fs.writeFileSync(HERE+'attribution.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
