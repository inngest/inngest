const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {lineage,fig,px,cy,arrow,tag,dot,ribbon,EV,C,W,LBL,PLOT} from './micro.mjs';
const E={};const D=(k,v)=>{E[k]=v;};

// ---- B. Inngest's own time (c14–c21) ------------------------------------
D('c14',{d:'The step waiting its turn. Thin, quiet, and it ends at the circle where your code starts.',
  svg:fig([{n:'a',segs:[['idle',0,34],['good',34,40]],note:'+34ms queued  40ms'}],'','queued')});
D('c15',{d:'Inngest working out what to run next. A discovery bar only appears where the request was a separate execution &mdash; a fan-out, or the first step of a run. Where it produced a single step it is rolled into that step instead.',
  svg:fig([{n:'req + a',segs:[['disc',0,26],['idle',26,6],['good',32,42]],noHalo:[26],note:'+26ms planning  42ms'},
           {n:'b',segs:[['idle',26,6],['good',32,34]],noHalo:[26],note:'34ms'}],'',
    '',ribbon(26,[cy(0),cy(1)]))});
D('c16',{d:'Being held by flow control is queue time with a reason, and the reason is worth its own colour: amber hatched, still not your compute, but distinguishable at a glance from a step simply waiting its turn.',
  svg:fig([{n:'a',segs:[['idle',0,10],['hold',10,52],['good',62,30]],note:'+6.3s held  30ms'}],
    tag(px(22),cy(0)-6,'concurrency: 1',C.mut),'concurrency hold')});
D('c16b',{d:'The same treatment covers every flow-control hold — throttle, rate limit, debounce — because they are the same fact about the run: it was ready and Inngest chose not to start it yet.',
  svg:fig([{n:'a',segs:[['idle',0,6],['hold',6,30],['good',36,24]],note:'throttled'},
           {n:'b',segs:[['idle',0,6],['hold',6,54],['good',60,24]],note:'rate limited'},
           {n:'c',segs:[['idle',0,6],['hold',6,18],['good',24,20]],note:'debounced'}],'','flow control holds')});
D('c31',{d:'A step.sendEvent() that started runs points out of this run: a short spur, a hollow ring and a count. Hollow because those runs are not in this trace.',
  svg:fig([{n:'notify',segs:[['idle',0,6],['good',6,36]]}],'','outbound lineage','',
    {scale:1.3, over:(X,CY)=>lineage(X(42),CY(0),2)})});
D('c17',{d:'Latency is the same substance as queueing and is named the same way. Three shades of grey would be three things to learn for no gain.',
  svg:fig([{n:'a',segs:[['idle',0,18],['good',18,54]],note:'+18ms latency  54ms'}],'','system latency')});
D('c18',{d:'Platform rows keep their own row and the platform colour. They are not your compute and never enter the Run row profile.',
  svg:fig([{n:'last step',segs:[['good',4,30]]},
    {n:'Finalization',segs:[['idle',34,20],['good',54,8]],dim:.6,note:'platform'}],'','finalization')});
D('c19',{d:'Everything Inngest did is thinner and quieter than your code. The eye lands on green and red first, every time.',
  svg:fig([{n:'a',segs:[['idle',0,28.0],['good',28,46]],note:'46ms of 74ms is yours'}],'','second-class platform time')});
D('c20',{d:'No unexplained gaps. Every millisecond between the row start and its resolution belongs to a named segment — here a discovery request, the queue, a failed attempt, its backoff, the queue again, and the attempt that worked.',
  svg:fig([{n:'a',segs:[['idle',0,34.0],['bad',34,28],['backoff',62,10],['idle',72,4],['good',76,18]],note:'fully accounted'}],'','gaps filled')});
D('c21',{d:'The 26ms between one step ending and the next request starting is drawn once — as the arrow and as the interval, the same object.',
  svg:(()=>{const rows=[{n:'c',segs:[['good',4,36]]},{n:'d',segs:[['idle',52,16],['good',70,22]]}];
    return fig(rows,arrow(px(40),cy(0),px(52),cy(1))+tag(px(41),cy(0)-6,'26ms',C.acc),'the connector is the interval');})()});

// ---- C. Step outcomes (c22–c30) -----------------------------------------
D('c22',{d:'Green bar, green resolution circle. The baseline everything else is read against.',
  svg:fig([{n:'a',segs:[['idle',0,10],['good',10,50]],note:'50ms'}],'','success')});
D('c23',{d:'The step is red and so is the run. The failure is the last thing on the row, so nothing after it implies recovery.',
  svg:fig([{n:'doomed',segs:[['idle',0,10],['bad',10,26]],note:'failed  26ms'},
    {n:'Run',segs:[['idle',0,36],['bad',36,4]],dim:.6,note:'FAILED'}],'','run-ending failure')});
D('c24',{d:'A red step inside a green run. The row is red at its own resolution; the Run row stays green because userland caught it.',
  svg:fig([{n:'caught',segs:[['idle',0,8],['bad',8,22]],note:'failed  22ms'},
    {n:'after',segs:[['idle',30,6],['good',36,34]],note:'34ms'}],'','caught failure')});
D('c25',{d:'Attempt one is red, the circle after it is neutral and means retrying, and only the resolution is green. The recovery is the shape of the row.',
  svg:fig([{n:'flaky',segs:[['idle',0,8],['bad',8,20],['backoff',28,18],['idle',46,4],['good',50,30]],note:'2 attempts  recovered'}],'','retry into success')});
D('c26',{d:'One red row does not tint its neighbours or the level. Failure is per-row and per-slice, never inherited.',
  svg:fig([{n:'ok-branch',segs:[['idle',0,8],['good',8,40]],note:'40ms'},
    {n:'bad-branch',segs:[['idle',0,8],['bad',8,26]],note:'failed'},
    {n:'after',segs:[['idle',50,6],['good',56,30]],note:'30ms'}],'','one red node stays local')});
D('c27',{d:'Cancelled is its own state — neither green nor red. <code>b</code> was still executing when the run was cut, so its bar is the running amber and it ends on the square: stopped from outside, never resolved.',
  svg:fig([
    {n:'a',segs:[['idle',0,8],['good',8,30]],note:'30ms'},
    {n:'b',segs:[['idle',0,8],['running',8,44]],dots:[{p:0,c:EV.queued},{p:8,c:EV.started},{p:52,c:EV.cancelled}],note:'cancelled'},
  ],'','a step cancelled mid-execution')});
D('c28',{d:'A <code>waitForEvent</code> still open when the run was cut. It was undecided, so the bar is the amber wait, and it ends on the square rather than a resolution.',
  svg:fig([
    {n:'waiting',segs:[['idle',0,6],['wait',6,52]],dots:[{p:0,c:EV.queued},{p:6,c:EV.started},{p:58,c:EV.cancelled}],note:'cancelled while open'},
  ],'','a wait cancelled while open')});
D('c29',{d:'No end time to draw to. The bar is the in-progress blue and has no terminal circle because nothing has resolved.',
  svg:fig([
    {n:'landed',segs:[['idle',0,8],['good',8,34]],note:'34ms'},
    {n:'still going',segs:[['idle',0,8],['running',8,86]],note:'running'},
  ],'','a step still running')});
D('c30',{d:'Every attempt failed. Three red bars, neutral retry circles between them, and a red resolution — the only circle that ever goes red.',
  svg:fig([{n:'doomed',segs:[['bad',0,14],['backoff',14,8],['idle',22,2],['bad',24,16],['backoff',40,12],['idle',52,2],['bad',54,20]],note:'3 attempts  failed'}],'','retries exhausted')});

fs.writeFileSync(HERE+'items-bc.json',JSON.stringify(E));
console.log('items',Object.keys(E).length);

// Re-emit in the frames shape, adding hover fragments where the decision only
// shows up on interaction.
import {fig as F2, px as X, cy as Y, arrow as AR, tag as TG, C as CC} from './micro.mjs';
const OUT={};
for(const [k,v] of Object.entries(E)) OUT[k]={d:v.d,frames:[{l:'at rest',svg:v.svg}]};

OUT.c21.frames=[
 {l:'at rest',svg:F2([{n:'c',segs:[['good',4,36]]},{n:'d',segs:[['idle',52,16],['good',70,22]]}],'','gap at rest')},
 {l:'hovering c',svg:F2([{n:'c',segs:[['good',4,36]],sel:true},{n:'d',segs:[['idle',52,16],['good',70,22]]}],
   AR(X(40),Y(0),X(52),Y(1))+TG(X(41),Y(0)-5,'26ms',CC.acc),'gap on hover')},
];
OUT.c26.frames.push({l:'hovering bad-branch',svg:F2([
  {n:'ok-branch',segs:[['idle',0,8],['good',8,40]],dim:.15},
  {n:'bad-branch',segs:[['idle',0,8],['bad',8,26]],note:'failed',sel:true},
  {n:'after',segs:[['idle',50,6],['good',56,30]],dim:.15}],
  '','failure stays local on hover')});
OUT.c25.frames.push({l:'hovering the backoff',svg:F2([
  {n:'flaky',segs:[['idle',0,8],['bad',8,20],['backoff',28,18],['idle',46,4],['good',50,30]],sel:true}],
  TG(X(30),Y(0)-6,'waited 2s before retrying',CC.mut),'backoff on hover')});

fs.writeFileSync(HERE+'items-bc.json',JSON.stringify(OUT));
console.log('bc frames',Object.values(OUT).reduce((n,x)=>n+x.frames.length,0));
