const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {lineage,fig,px,cy,arrow,tag,dot,ribbon,EV,C,W,LBL,PLOT} from './micro.mjs';
import {setFrame} from './micro.mjs'; setFrame(true);
const E={};const D=(k,v)=>{E[k]=v;};

// ---- B. Inngest's own time (c14–c21) ------------------------------------
D('c14',{d:'The step waiting its turn. Thin, quiet, and it ends at the circle where your code starts.',
  svg:fig([{n:'a',at:[['queued',0],['started',340],['ok',740]],note:'+340ms queued · 400ms'}],'','queued','',{trimLead:false})});
D('c15',{d:'Inngest working out what to run next. A discovery bar only appears where the request was a separate execution &mdash; a fan-out, or the first step of a run. Where it produced a single step it is rolled into that step instead.',
  svg:fig([{n:'req + a',at:[['queued',0],['started',60,'disc'],['ok',260],['planned',260],['started',320],['ok',740]],reported:1,note:'+26ms planning  42ms'},
           {n:'b',at:[['planned',260],['started',320],['ok',660]],note:'34ms'}],'',
    '')});
D('c16',{d:'Being held by flow control is queue time with a reason, and the reason is worth its own colour: amber hatched, still not your compute, but distinguishable at a glance from a step simply waiting its turn.',
  svg:fig([{n:'a',at:[['queued',0],['held',90],['started',1400],['ok',1700]],note:'+1.4s held · 300ms'}],
    '','concurrency hold','',{trimLead:false, linear:true})});
D('c16b',{d:'The same treatment covers every flow-control hold — throttle, rate limit, debounce — because they are the same fact about the run: it was ready and Inngest chose not to start it yet.',
  svg:fig([{n:'a',at:[['queued',0],['held',60],['started',360],['ok',600]],note:'throttled'},
           {n:'b',at:[['queued',0],['held',60],['started',600],['ok',840]],note:'rate limited'},
           {n:'c',at:[['queued',0],['held',60],['started',240],['ok',440]],note:'debounced'}],'','flow control holds')});
D('c31',{d:'A step.sendEvent() that started runs points out of this run: a short spur, a hollow ring and a count. Hollow because those runs are not in this trace.',
  svg:fig([{n:'notify',at:[['queued',0],['started',60],['ok',420]],lineage:2}],'','outbound lineage','',
    {scale:1.3})});
D('c18',{d:'Platform rows keep their own row and the platform colour. They are not your compute and never enter the Run row profile.',
  svg:fig([{n:'a',at:[['started',40],['ok',340]]}],'','finalization','',{trimLead:false})});
D('c20',{d:'No unexplained gaps. Every millisecond between the row start and its resolution belongs to a named segment — here a discovery request, the queue, a failed attempt, its backoff, the queue again, and the attempt that worked.',
  svg:fig([{n:'a',at:[['queued',0],['started',340],['retry',620],['queued',720],['started',760],['ok',940]],note:'fully accounted'}],'','gaps filled')});
D('c21',{d:'The 26ms between one step ending and the next request starting is drawn once — as the arrow and as the interval, the same object.',
  svg:(()=>{const rows=[{n:'a',at:[['started',40],['ok',400]]},{n:'b',at:[['queued',520],['started',700],['ok',920]]}];
    return fig(rows,'','the connector is the interval');})()});

// ---- C. Step outcomes (c22–c30) -----------------------------------------
D('c23',{d:'The step is red and so is the run. The failure is the last thing on the row, so nothing after it implies recovery.',
  svg:fig([{n:'a',at:[['queued',0],['started',100],['failed',360]],note:'failed  26ms'}
  ],'','run-ending failure')});
D('c24',{d:'A red step inside a green run. The row is red at its own resolution; the Run row stays green because userland caught it.',
  svg:fig([{n:'caught',at:[['queued',0],['started',80],['failed',300]],note:'failed  22ms'},
    {n:'after',at:[['queued',300],['started',360],['ok',700]],note:'34ms'}],'','caught failure')});
D('c25',{d:'Attempt one is red, the circle after it is neutral and means retrying, and only the resolution is green. The recovery is the shape of the row.',
  svg:fig([{n:'a',at:[['queued',0],['started',80],['retry',360],['queued',1360],['started',1420],['ok',1760]],note:'2 attempts  recovered'}],'','retry into success')});
D('c26',{d:'One red row does not tint its neighbours or the level. Failure is per-row and per-slice, never inherited.',
  svg:fig([{n:'ok-branch',at:[['queued',0],['started',80],['ok',480]],note:'40ms'},
    {n:'bad-branch',at:[['queued',0],['started',80],['failed',340]],note:'failed'},
    {n:'after',at:[['queued',500],['started',560],['ok',860]],note:'30ms'}],'','one red node stays local')});
D('c27',{d:'Cancelled is its own state — neither green nor red. <code>b</code> was still executing when the run was cut, so its bar is the running amber and it ends on the square: stopped from outside, never resolved.',
  svg:fig([
    {n:'a',at:[['queued',0],['started',80],['ok',380]],note:'30ms'},
    {n:'b',at:[['queued',0],['started',80],['cancelled',520]],note:'cancelled'},
  ],'','a step cancelled mid-execution')});
D('c28',{d:'A <code>waitForEvent</code> still open when the run was cut. It was undecided, so the bar is the amber wait, and it ends on the square rather than a resolution.',
  svg:fig([
    {n:'w',at:[['queued',0],['started',60],['cancelled',5800]],kind:'wait',note:'cancelled while open'},
  ],'','a wait cancelled while open','',{linear:true})});
D('c29',{d:'No end time to draw to. The bar is the in-progress blue and has no terminal circle because nothing has resolved.',
  svg:fig([
    {n:'landed',at:[['queued',0],['started',80],['ok',420]],note:'34ms'},
    {n:'still going',at:[['queued',0],['started',80]],end:940,note:'running'},
  ],'','a step still running')});
D('c30',{d:'Every attempt failed. Three red bars, neutral retry circles between them, and a red resolution — the only circle that ever goes red.',
  svg:fig([{n:'a',at:[['started',0],['retry',240],['queued',1240],['started',1300],['retry',1560],['queued',3560],['started',3620],['failed',3880]],note:'3 attempts  failed'}],'','retries exhausted')});

fs.writeFileSync(HERE+'items-bc.json',JSON.stringify(E));
console.log('items',Object.keys(E).length);

// Re-emit in the frames shape, adding hover fragments where the decision only
// shows up on interaction.
import {fig as F2, px as X, cy as Y, arrow as AR, tag as TG, C as CC} from './micro.mjs';
const OUT={};
for(const [k,v] of Object.entries(E)) OUT[k]={d:v.d,frames:[{l:'at rest',svg:v.svg}]};

OUT.c21.frames=[
 {l:'at rest',svg:F2([{n:'a',at:[['started',40],['ok',400]]},{n:'b',at:[['queued',520],['started',700],['ok',920]]}],'','gap at rest')},
 {l:'hovering a',svg:F2([{n:'a',at:[['started',40],['ok',400]],sel:true},{n:'b',at:[['queued',520],['started',700],['ok',920]]}],
   AR(X(40),Y(0),X(52),Y(1))+TG(X(41),Y(0)-5,'26ms',CC.acc),'gap on hover')},
];
OUT.c26.frames.push({l:'hovering bad-branch',svg:F2([
  {n:'ok-branch',at:[['queued',0],['started',80],['ok',480]],dim:.15},
  {n:'bad-branch',at:[['queued',0],['started',80],['failed',340]],note:'failed',sel:true},
  {n:'after',at:[['queued',500],['started',560],['ok',860]],dim:.15}],
  '','failure stays local on hover')});
OUT.c25.frames.push({l:'hovering the backoff',svg:F2([
  {n:'a',at:[['queued',0],['started',80],['retry',360],['queued',1360],['started',1420],['ok',1760]],sel:true}],
  TG(X(30),Y(0)-6,'waited 2s before retrying',CC.mut),'backoff on hover')});

fs.writeFileSync(HERE+'items-bc.json',JSON.stringify(OUT));
console.log('bc frames',Object.values(OUT).reduce((n,x)=>n+x.frames.length,0));
