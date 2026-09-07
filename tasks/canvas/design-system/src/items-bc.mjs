const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {lineage,fig,px,cy,arrow,tag,dot,ribbon,EV,C,W,LBL,PLOT} from './micro.mjs';
import {setFrame} from './micro.mjs'; setFrame(true);
const E={};const D=(k,v)=>{E[k]=v;};

// ---- B. Inngest's own time (c14–c21) ------------------------------------
D('c14',{d:'The step waiting its turn. Thin, quiet, and it ends at the circle where your code starts.',
  svg:fig([{n:'a',at:[['queued',0],['started',34],['ok',74]],note:'+34ms queued  40ms'}],'','queued')});
D('c15',{d:'Inngest working out what to run next. A discovery bar only appears where the request was a separate execution &mdash; a fan-out, or the first step of a run. Where it produced a single step it is rolled into that step instead.',
  svg:fig([{n:'req + a',at:[['queued',0],['started',6,'disc'],['ok',26],['queued',26],['started',32],['ok',74]],reported:1,note:'+26ms planning  42ms'},
           {n:'b',at:[['queued',26],['started',32],['ok',66]],note:'34ms'}],'',
    '')});
D('c16',{d:'Being held by flow control is queue time with a reason, and the reason is worth its own colour: amber hatched, still not your compute, but distinguishable at a glance from a step simply waiting its turn.',
  svg:fig([{n:'a',at:[['queued',0],['held',10],['started',62],['ok',92]],note:'+6.3s held  30ms'}],
    tag(px(22),cy(0)-6,'concurrency: 1',C.mut),'concurrency hold')});
D('c16b',{d:'The same treatment covers every flow-control hold — throttle, rate limit, debounce — because they are the same fact about the run: it was ready and Inngest chose not to start it yet.',
  svg:fig([{n:'a',at:[['queued',0],['held',6],['started',36],['ok',60]],note:'throttled'},
           {n:'b',at:[['queued',0],['held',6],['started',60],['ok',84]],note:'rate limited'},
           {n:'c',at:[['queued',0],['held',6],['started',24],['ok',44]],note:'debounced'}],'','flow control holds')});
D('c31',{d:'A step.sendEvent() that started runs points out of this run: a short spur, a hollow ring and a count. Hollow because those runs are not in this trace.',
  svg:fig([{n:'notify',at:[['queued',0],['started',6],['ok',42]],lineage:2}],'','outbound lineage','',
    {scale:1.3})});
D('c17',{d:'Latency is the same substance as queueing and is named the same way. Three shades of grey would be three things to learn for no gain.',
  svg:fig([{n:'a',at:[['queued',0],['started',18],['ok',72]],note:'+18ms latency  54ms'}],'','system latency')});
D('c18',{d:'Platform rows keep their own row and the platform colour. They are not your compute and never enter the Run row profile.',
  svg:fig([{n:'last step',at:[['started',4],['ok',34]]},
    {n:'Finalization',at:[['queued',34],['started',54],['ok',62]],dim:.6,note:'platform'}],'','finalization')});
D('c19',{d:'Everything Inngest did is thinner and quieter than your code. The eye lands on green and red first, every time.',
  svg:fig([{n:'a',at:[['queued',0],['started',28],['ok',74]],note:'46ms of 74ms is yours'}],'','second-class platform time')});
D('c20',{d:'No unexplained gaps. Every millisecond between the row start and its resolution belongs to a named segment — here a discovery request, the queue, a failed attempt, its backoff, the queue again, and the attempt that worked.',
  svg:fig([{n:'a',at:[['queued',0],['started',34],['retry',62],['queued',72],['started',76],['ok',94]],note:'fully accounted'}],'','gaps filled')});
D('c21',{d:'The 26ms between one step ending and the next request starting is drawn once — as the arrow and as the interval, the same object.',
  svg:(()=>{const rows=[{n:'c',at:[['started',4],['ok',40]]},{n:'d',at:[['queued',52],['started',70],['ok',92]]}];
    return fig(rows,arrow(px(40),cy(0),px(52),cy(1))+tag(px(41),cy(0)-6,'26ms',C.acc),'the connector is the interval');})()});

// ---- C. Step outcomes (c22–c30) -----------------------------------------
D('c22',{d:'Green bar, green resolution circle. The baseline everything else is read against.',
  svg:fig([{n:'a',at:[['queued',0],['started',10],['ok',60]],note:'50ms'}],'','success')});
D('c23',{d:'The step is red and so is the run. The failure is the last thing on the row, so nothing after it implies recovery.',
  svg:fig([{n:'doomed',at:[['queued',0],['started',10],['failed',36]],note:'failed  26ms'}
  ],'','run-ending failure')});
D('c24',{d:'A red step inside a green run. The row is red at its own resolution; the Run row stays green because userland caught it.',
  svg:fig([{n:'caught',at:[['queued',0],['started',8],['failed',30]],note:'failed  22ms'},
    {n:'after',at:[['queued',30],['started',36],['ok',70]],note:'34ms'}],'','caught failure')});
D('c25',{d:'Attempt one is red, the circle after it is neutral and means retrying, and only the resolution is green. The recovery is the shape of the row.',
  svg:fig([{n:'flaky',at:[['queued',0],['started',8],['retry',28],['queued',46],['started',50],['ok',80]],note:'2 attempts  recovered'}],'','retry into success')});
D('c26',{d:'One red row does not tint its neighbours or the level. Failure is per-row and per-slice, never inherited.',
  svg:fig([{n:'ok-branch',at:[['queued',0],['started',8],['ok',48]],note:'40ms'},
    {n:'bad-branch',at:[['queued',0],['started',8],['failed',34]],note:'failed'},
    {n:'after',at:[['queued',50],['started',56],['ok',86]],note:'30ms'}],'','one red node stays local')});
D('c27',{d:'Cancelled is its own state — neither green nor red. <code>b</code> was still executing when the run was cut, so its bar is the running amber and it ends on the square: stopped from outside, never resolved.',
  svg:fig([
    {n:'a',at:[['queued',0],['started',8],['ok',38]],note:'30ms'},
    {n:'b',at:[['queued',0],['started',8]],end:52,dots:[{p:0,c:EV.queued},{p:8,c:EV.started},{p:52,c:EV.cancelled}],note:'cancelled'},
  ],'','a step cancelled mid-execution')});
D('c28',{d:'A <code>waitForEvent</code> still open when the run was cut. It was undecided, so the bar is the amber wait, and it ends on the square rather than a resolution.',
  svg:fig([
    {n:'waiting',at:[['queued',0],['started',6]],kind:'wait',end:58,dots:[{p:0,c:EV.queued},{p:6,c:EV.started},{p:58,c:EV.cancelled}],note:'cancelled while open'},
  ],'','a wait cancelled while open')});
D('c29',{d:'No end time to draw to. The bar is the in-progress blue and has no terminal circle because nothing has resolved.',
  svg:fig([
    {n:'landed',at:[['queued',0],['started',8],['ok',42]],note:'34ms'},
    {n:'still going',at:[['queued',0],['started',8]],end:94,note:'running'},
  ],'','a step still running')});
D('c30',{d:'Every attempt failed. Three red bars, neutral retry circles between them, and a red resolution — the only circle that ever goes red.',
  svg:fig([{n:'doomed',at:[['started',0],['retry',14],['queued',22],['started',24],['retry',40],['queued',52],['started',54],['failed',74]],note:'3 attempts  failed'}],'','retries exhausted')});

fs.writeFileSync(HERE+'items-bc.json',JSON.stringify(E));
console.log('items',Object.keys(E).length);

// Re-emit in the frames shape, adding hover fragments where the decision only
// shows up on interaction.
import {fig as F2, px as X, cy as Y, arrow as AR, tag as TG, C as CC} from './micro.mjs';
const OUT={};
for(const [k,v] of Object.entries(E)) OUT[k]={d:v.d,frames:[{l:'at rest',svg:v.svg}]};

OUT.c21.frames=[
 {l:'at rest',svg:F2([{n:'c',at:[['started',4],['ok',40]]},{n:'d',at:[['queued',52],['started',70],['ok',92]]}],'','gap at rest')},
 {l:'hovering c',svg:F2([{n:'c',at:[['started',4],['ok',40]],sel:true},{n:'d',at:[['queued',52],['started',70],['ok',92]]}],
   AR(X(40),Y(0),X(52),Y(1))+TG(X(41),Y(0)-5,'26ms',CC.acc),'gap on hover')},
];
OUT.c26.frames.push({l:'hovering bad-branch',svg:F2([
  {n:'ok-branch',at:[['queued',0],['started',8],['ok',48]],dim:.15},
  {n:'bad-branch',at:[['queued',0],['started',8],['failed',34]],note:'failed',sel:true},
  {n:'after',at:[['queued',50],['started',56],['ok',86]],dim:.15}],
  '','failure stays local on hover')});
OUT.c25.frames.push({l:'hovering the backoff',svg:F2([
  {n:'flaky',at:[['queued',0],['started',8],['retry',28],['queued',46],['started',50],['ok',80]],sel:true}],
  TG(X(30),Y(0)-6,'waited 2s before retrying',CC.mut),'backoff on hover')});

fs.writeFileSync(HERE+'items-bc.json',JSON.stringify(OUT));
console.log('bc frames',Object.values(OUT).reduce((n,x)=>n+x.frames.length,0));
