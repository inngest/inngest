const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,dot,tag,EV,C,W,LBL,PLOT,ROW,TOP} from './micro.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";
const O={};

const KEY=[
 ['queued',   'a discovery request is queued — the step is not known yet', EV.queued, 'still going'],
 ['planned',  'a step someone already planned is queued',                  EV.ribbon, 'still going'],
 ['started',  'the execution began',                                       EV.started,'still going'],
 ['retry',    'that attempt failed; another is coming',  EV.retry,     'still going'],
 ['ok',       'succeeded',                               EV.ok,        'done'],
 ['failed',   'failed for good',                         EV.failed,    'done'],
 ['timeout',  'nothing matched in time',                 EV.timeout,   'done'],
 ['cancelled','stopped from outside',                    EV.cancelled, 'done'],
];
O.key=(()=>{
  const groups=[['still going — hollow',KEY.filter(r=>r[3]==='still going')],
                ['done — filled',KEY.filter(r=>r[3]!=='still going')]];
  let y=12,s='';
  for(const [title,rows] of groups){
    s+=`<text x="8" y="${y}" ${MONO} font-size="7" fill="${C.mut}" opacity=".65">${title}</text>`;
    y+=11;
    for(const [n,d,c] of rows){
      s+=dot(18,y,c)+`<text x="30" y="${y+2.5}" ${MONO} font-size="7.5" fill="${C.mut}">${n}</text>`
       +`<text x="90" y="${y+2.5}" ${MONO} font-size="7" fill="${C.mut}" opacity=".85">${d}</text>`;
      y+=15;
    }
    y+=6;
  }
  return `<svg viewBox="0 0 ${W} ${y}" role="img" aria-label="The circle kinds">`+s+`</svg>`;
})();

// Circles must read on a bar of their own colour.
O.halo=fig([
  {n:'on green',segs:[['good',0,60]]},
  {n:'on red',segs:[['bad',0,60]]},
  {n:'on blue',segs:[['disc',0,60]]},
],'','circles on same-coloured bars');

// The backoff is knowable, so draw it.
O.backoff=(()=>{
  const segs=[['idle',0,6],['bad',6,10],['backoff',16,10],['idle',26,4],['bad',30,10],['backoff',40,20],['idle',60,4],['bad',64,10],['backoff',74,14],['idle',88,4],['good',92,8]];
  return fig([{n:'flaky',segs}],'','backoff between attempts');
})();

O.applied=fig([
  {n:'a (fan-out)',segs:[['disc',0,14],['idle',14,10],['good',24,34]],note:'succeeded'},
  {n:'flaky',segs:[['idle',0,8],['bad',8,16],['backoff',24,14],['idle',38,4],['good',42,26]],note:'recovered'},
  {n:'doomed',segs:[['idle',0,8],['bad',8,18],['backoff',26,8],['idle',34,4],['bad',38,20]],note:'failed'},
],'','the vocabulary applied');

O.waits=fig([
  {n:'matched',segs:[['idle',0,6],['waitok',6,44]],note:'event arrived'},
  {n:'timed out',segs:[['idle',0,6],['waitout',6,64]],note:'nothing matched'},
  {n:'cancelled',segs:[['idle',0,6],['wait',6,52]],dots:[{p:0,c:EV.queued},{p:6,c:EV.started},{p:58,c:EV.cancelled}],note:'run cut'},
],'','three ways a wait can end');

fs.writeFileSync(HERE+'vocab.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
