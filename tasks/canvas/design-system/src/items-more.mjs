const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,tag,axis,groupRow,arrow,dot,EV,C,W,LBL,PLOT,ROW,TOP} from './micro.mjs';
const E={}; const D=(k,v)=>{E[k]=v;};
const DIM=.22;

// ---- Waiting ------------------------------------------------------------

D('w1',{d:'A wait that matched. The bar is blue while it is open because nothing has decided it, and turns green at the mark where the event arrived.',
  svg:fig([
    {n:'a',              segs:[['good',0,10]]},
    {n:'waitForEvent',   segs:[['disc',10,2],['waitok',12,52]]},
    {n:'b',              segs:[['disc',64,2],['idle',66,4],['good',70,16]]},
  ],'','a wait that matched')});

D('w2',{d:'A wait that expired without a match. The run carries on: the timeout is a result the function can act on, so the row goes grey and the next step is reported as normal.',
  svg:fig([
    {n:'a',              segs:[['good',0,10]]},
    {n:'waitForEvent',   segs:[['disc',10,2],['waitout',12,52]]},
    {n:'fallback',       segs:[['disc',64,2],['idle',66,4],['good',70,16]]},
  ],'','a wait that timed out, run continues')});

D('w3',{d:'A wait inside a fan-out. It sits on the same axis as its siblings and holds the level open: the request that collects them cannot start until the slowest resolves, which the queue time on the next row reports.',
  svg:fig([
    {n:'req + a', segs:[['disc',0,8],['good',8,20]]},
    {n:'b',       segs:[['good',8,14]]},
    {n:'wait c',  segs:[['waitok',8,54]]},
    {n:'d',       segs:[['disc',62,2],['idle',64,4],['good',68,18]]},
  ],'','a wait holding a level open','',{rib:{x:8,rows:[0,1,2]}})});

D('w4',{d:'Waiting and failing must not read alike. One is hatched and never red; the other is solid and red, and only its final attempt takes a filled mark.',
  svg:fig([
    {n:'waiting',  segs:[['wait',0,60]],dots:[{p:0,c:EV.queued},{p:0,c:EV.started}]},
    {n:'failing',  segs:[['bad',0,26],['backoff',26,14],['idle',40,4],['bad',44,26]]},
  ],'','the two must not be confused')});

// ---- Time & the axis ----------------------------------------------------

D('t1',{d:'An idle gap that dwarfs the work is compressed and marked as a break. The axis is the only thing that changes: every duration on the row is still wall clock.',
  svg:fig([
    {n:'a', segs:[['good',0,8]]},
    {n:'sleep 7d', segs:[['waitok',8,50]]},
    {n:'b', segs:[['disc',58,2],['idle',60,3],['good',63,12]]},
  ],axis(cy(2)+13,[[0,'0ms'],[8,'+41ms'],[58,'+7d'],[75,'+7d 62ms']],{brk:[[10,56]]}),
    'a compressed idle gap','',{margin:0,pad:34})});

D('t2',{d:'Seven days elapsed, 62ms executing. Reading the fill alone tells you that before you have read a number, which is the point of the height rule.',
  svg:fig([
    {run:true, end:100, intervals:[{a:0,b:1.2,ok:true},{a:58,b:59,ok:true},{a:63,b:64.5,ok:true}], resolvedAt:100, resolvedAs:EV.ok},
    {n:'a', segs:[['good',0,1.2]]},
    {n:'sleep 7d', segs:[['waitok',1.2,56.8]]},
    {n:'b', segs:[['disc',58,1],['idle',59,4],['good',63,1.5]]},
  ],'','62ms of execution inside seven days')});

D('t3',{d:'Two tiers of axis label. The coarse tier carries what the run crossed, the fine tier carries offsets inside it, so a run measured in days keeps its resolution without a second axis.',
  svg:fig([
    {n:'a', segs:[['good',0,10]]},
    {n:'b', segs:[['idle',10,40],['good',50,36]]},
  ],axis(cy(1)+13,[[0,'0'],[25,'+6h'],[50,'+12h'],[75,'+18h'],[100,'+24h']],
        {tier2:[[0,'Mar 4'],[50,'Mar 5']]}),'two-tier axis labels','',{pad:40})});

D('t4',{d:'A step too short to draw is still drawn. It gets a minimum width so it can be pointed at, and the number beside it is the real one: the drawing rounds, the reported duration does not.',
  svg:fig([
    {n:'12ms step', segs:[['good',0,0.4]]},
    {n:'2s sleep',  segs:[['waitok',0.4,96]]},
    {n:'10ms step', segs:[['disc',96.4,0.2],['good',96.6,0.4]]},
  ],tag(px(2),cy(0)+2.5,'12ms')+tag(px(98.5),cy(2)+2.5,'10ms'),'sub-pixel steps drawn honestly')});

// ---- Scale --------------------------------------------------------------

D('s1',{d:'Five hundred sequential steps do not become five hundred rows. Repetition with one shape collapses to a single row that reports the count and draws where each member ran.',
  svg:(()=>{
    const members=[]; for(let i=0;i<36;i++) members.push([4+i*2.1, 1.3, 'good']);
    return fig([
      {n:'setup', segs:[['good',0,3]]},
      {n:'', segs:[]},
      {n:'teardown', segs:[['idle',88,2],['good',90,8]]},
    ], groupRow(1,{n:'× 500 fetch',x:3,w:79,members,note:'500 · 0 failed'}),
      'five hundred steps, one row');
  })()});

D('s2',{d:'The same collapse for a wide fan-out. The envelope is the level and the ticks are the members, so the stagger of a concurrency limit is visible without expanding anything.',
  svg:(()=>{
    const members=[]; for(let i=0;i<12;i++) members.push([6+i*2.9, 21-i*0.6, 'good']);
    return fig([
      {n:'req + ×12', segs:[['disc',0,4]]},
      {n:'', segs:[]},
      {n:'collect', segs:[['disc',80,2],['idle',82,3],['good',85,12]]},
    ], groupRow(1,{n:'× 12 worker',x:5,w:70,members,note:'12 · staggered'}),
      'a wide fan-out, collapsed');
  })()});

D('s3',{d:'A cluster of failures two thirds through a long run is a red smear in the strip above the trace, findable in the first screenful without scrolling or interacting.',
  svg:(()=>{
    let s='';
    for(let i=0;i<96;i++){
      const bad = i>60 && i<70;
      const h = bad?9:4+((i*7)%4);
      s+=`<rect x="${LBL+i*(PLOT/96)}" y="${cy(0)-h/2}" width="${PLOT/96-0.6}" height="${h}" rx="0.6" fill="${bad?C.bad:C.good}" opacity="${bad?1:.5}"/>`;
    }
    return fig([{n:'',segs:[]},{n:'',segs:[]}], s+tag(px(63),cy(1)+2,'a failure cluster, 500 steps in'),
      'a failure cluster in the density strip');
  })()});

D('s4',{d:'Whatever the step count, the trace draws about forty rows at rest. Everything past that is inside a collapsed group, and every collapsed group can be expanded where you are standing.',
  svg:fig([
    {n:'first', segs:[['good',0,6]]},
    {n:'× 40 batch', segs:[['idle',6,2],['good',8,60]],note:''},
    {n:'  ↳ member 7', segs:[['idle',20,1],['good',21,6]]},
    {n:'  ↳ member 8', segs:[['idle',24,1],['good',25,5]]},
    {n:'last', segs:[['disc',70,2],['idle',72,3],['good',75,12]]},
  ],tag(px(69),cy(1)+2.5,'× 40'),'one group expanded in place')});

// ---- Naming & identity --------------------------------------------------

D('n1',{d:'The same step name on two branches. The name is not the identity: rows are distinguished by which request reported them, which the ribbon already shows.',
  svg:fig([
    {n:'req + fetch', segs:[['disc',0,7],['good',7,26]]},
    {n:'fetch',       segs:[['good',7,18]]},
    {n:'save',        segs:[['disc',33,2],['idle',35,3],['good',38,20]]},
  ],'','the same name twice','',{rib:{x:7,rows:[0,1]}})});

D('n2',{d:'The SDK’s :1 and :2 suffixes are not stable under parallelism, so nothing in the drawing depends on them. They are shown as text on the row and used for nothing else.',
  svg:fig([
    {n:'fetch:1', segs:[['good',6,20]]},
    {n:'fetch:2', segs:[['good',6,30]]},
  ],tag(px(38),cy(0)+2.5,'suffix is reported, not relied on'),'unstable suffixes')});

// ---- Interaction --------------------------------------------------------

D('i1',{d:'Three tiers of attention on hover: the row itself, what caused it, and everything else. Nothing is removed, only quietened, so the shape of the run is still readable while you read one row.',
  svg:fig([
    {n:'a',  segs:[['good',0,18]],dim:1},
    {n:'b',  segs:[['idle',18,4],['good',22,26]],sel:true},
    {n:'c',  segs:[['good',4,12]],dim:DIM},
    {n:'d',  segs:[['idle',48,3],['good',51,20]],dim:DIM},
  ],arrow(px(18),cy(0),px(22),cy(1)),'hovered, immediate cause, everything else')});

D('i2',{d:'The strip above the run is a minimap of the same trace, at the same order and the same colours. Dragging it sets the viewport; it never shows a different set of steps from the one below it.',
  svg:(()=>{
    let s='';
    const rows=[[0,18,'good'],[18,30,'good'],[48,10,'bad'],[58,28,'good']];
    rows.forEach((r,i)=>{ s+=`<rect x="${px(r[0])}" y="${cy(0)-6+i*3}" width="${(r[1]/100)*PLOT}" height="2" rx="1" fill="${r[2]==='bad'?C.bad:C.good}" opacity=".8"/>`; });
    s+=`<rect x="${px(30)}" y="${cy(0)-8}" width="${(40/100)*PLOT}" height="16" rx="2" fill="none" stroke="${C.acc}" stroke-width="1" opacity=".8"/>`;
    return fig([{n:'minimap',segs:[]},{n:'',segs:[]},
      {n:'b', segs:[['idle',30,4],['good',34,26]]},
      {n:'c', segs:[['bad',60,10]]},
    ], s, 'the minimap mirrors the trace');
  })()});

// ---- Honesty ------------------------------------------------------------

D('h1',{d:'Where the trace cannot say which step caused a request, it declines rather than guessing. No cable is drawn and the row says so, which is a smaller cost than a confident wrong line.',
  svg:fig([
    {n:'a', segs:[['good',0,20]],dim:DIM},
    {n:'b', segs:[['good',4,24]],dim:DIM},
    {n:'c', segs:[['idle',28,4],['good',32,24]],sel:true},
  ],tag(px(58),cy(2)+2.5,'no reported parent'),'declining to attribute')});

D('h2',{d:'A row’s label and its drawing have to agree. If the bar is drawn across a wider interval than the step ran for, the number beside it names the interval that was widened, not the one the SDK reported.',
  svg:fig([
    {n:'a', segs:[['disc',0,3],['idle',3,8],['good',11,20]]},
  ],tag(px(33),cy(0)+2.5,'3ms reported · 31ms on the row'),'the label reconciles with the drawing')});

D('h3',{d:'Nothing is drawn that cannot be asked what it is. Every interval and every mark on every row decomposes into named parts on hover, which is also how this document proves it has no unexplained pixels.',
  svg:fig([
    {n:'req + a', segs:[['disc',0,8],['idle',8,4],['good',12,30]]},
  ],'','everything decomposes')});

fs.writeFileSync(HERE+'items-more.json',JSON.stringify(
  Object.fromEntries(Object.entries(E).map(([k,v])=>[k,{d:v.d,frames:[{l:'at rest',svg:v.svg}]}]))));
console.log('more',Object.keys(E).length);
