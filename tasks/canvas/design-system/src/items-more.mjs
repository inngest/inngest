const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,tag,axis,groupRow,arrow,dot,elastic,EV,C,W,LBL,PLOT,ROW,TOP} from './micro.mjs';
import {setFrame} from './micro.mjs'; setFrame(true);
const E={}; const D=(k,v)=>{E[k]=v;};
const DIM=.22;

// ---- Waiting ------------------------------------------------------------

D('w1',{d:'A wait that matched. The bar is blue while it is open because nothing has decided it, and turns green at the mark where the event arrived.',
  svg:fig([
    {n:'a',              segs:[['good',0,10]]},
    {n:'w',              segs:[['disc',10,2],['waitok',12,52]]},
    {n:'b',              segs:[['disc',64,2],['idle',66,4],['good',70,16]]},
  ],'','a wait that matched')});

D('w2',{d:'A wait that expired without a match. The run carries on: the timeout is a result the function can act on, so the row goes grey and the next step is reported as normal.',
  svg:fig([
    {n:'a',              segs:[['good',0,10]]},
    {n:'w',              segs:[['disc',10,2],['waitout',12,52]]},
    {n:'fallback',       segs:[['disc',64,2],['idle',66,4],['good',70,16]]},
  ],'','a wait that timed out, run continues')});

D('w3',{d:'A wait inside a fan-out. It sits on the same axis as its siblings and holds the level open: the request that collects them cannot start until the slowest resolves, which the queue time on the next row reports.',
  svg:fig([
    {n:'req + a', segs:[['idle',0,2.8],['disc',2.8,5.2],['good',8,20]]},
    {n:'b',       segs:[['good',8,14]]},
    {n:'c',       segs:[['waitok',8,54]]},
    {n:'d',       segs:[['disc',62,2],['idle',64,4],['good',68,18]]},
  ],'','a wait holding a level open','',{rib:{x:8,rows:[0,1,2]}})});

D('w4',{d:'Waiting and failing must not read alike. One is hatched and never red; the other is solid and red, and only its final attempt takes a filled mark.',
  svg:fig([
    {n:'nap',      segs:[['wait',0,60]],dots:[{p:0,c:EV.queued},{p:0,c:EV.started}]},
    {n:'a',        segs:[['bad',0,26],['backoff',26,14],['idle',40,4],['bad',44,26]]},
  ],'','the two must not be confused')});

// ---- Time & the axis ----------------------------------------------------

// Every figure in this section is LAID OUT by `elastic()` from real durations
// rather than placed by hand, so the rules it demonstrates are the rules that
// produced it. A figure that illustrates a layout rule by hand-placing bars is
// only a drawing of what we hope the rule does.
const NAP=7*864e5;

D('t1',{d:'Seven days of dead time, given four percent of the width. The threshold is low on purpose: if nothing is executing for more than a few percent of the run, that stretch is worth almost none of the space, and an hour and a week get the same few pixels. The cut takes the <em>middle</em> of the sleep, so the bar visibly begins, is torn, and resumes &mdash; it is that bar being compressed, not merely something happening between two rows. Only the drawing compresses: the sleep still reports 7d.',
  svg:(()=>{
    const B0=41+NAP, total=B0+62;
    const el=elastic(total,[[41,41+NAP]]);
    const g=(k,a,b)=>[k, el.at(a), el.at(b)-el.at(a)];
    const bd=el.bands[0];
    return fig([
      {n:'a',   segs:[g('good',0,41)]},
      {n:'nap', segs:[g('waitok',41,41+NAP)],note:'7d'},
      {n:'b',   segs:[g('idle',B0,B0+6),g('disc',B0+6,B0+16),g('idle',B0+16,B0+22),g('good',B0+22,total)]},
    ],'','seven days compressed to a band','',{margin:0,breaks:[[bd.p0,bd.p1,'7d']]});
  })()});

D('t1b',{d:'What the space left over is worth. Thirty seconds of work on one side of a compressed gap and ten on the other, so the remaining width splits 75/25 &mdash; which is the same thing as saying <strong>a second is the same number of pixels wherever it lands</strong>. Without that rule, compressing a gap would quietly rescale one half of the trace against the other and two spans either side of it would stop being comparable. It is also why several compressions need no special case: the arithmetic is total live width over total live time, applied everywhere.',
  svg:(()=>{
    const total=30+7*86400+10;
    const el=elastic(total,[[30,30+7*86400]]);
    const g=(k,a,b)=>[k, el.at(a), el.at(b)-el.at(a)];
    const bd=el.bands[0];
    return fig([
      {n:'left',  segs:[g('good',0,30)],note:'30s'},
      {n:'gap',   segs:[g('waitok',30,30+7*86400)]},
      {n:'right', segs:[g('good',30+7*86400,total)],note:'10s'},
    ],'','the leftover width splits by real duration','',{margin:0,breaks:[[bd.p0,bd.p1,'7d']]});
  })()});

D('t1c',{d:'Many compressions. A polling loop is a collapsed group, so the cuts fall <em>inside</em> one pair of rows rather than adding rows of their own. The bands share one budget: eight idle stretches do not spend the whole width on the parts where nothing happened, they thin instead, and below a width that can hold them a band drops its label, then its tear, leaving a marked line. The rules and the blur never go &mdash; they are what says <em>not to scale</em>. The polls between the cuts still share the one scale.',
  svg:(()=>{
    const POLL=4, WAIT=120;
    const polls=[], dead=[]; let t=0;
    for(let i=0;i<9;i++){
      polls.push([t,t+POLL]); t+=POLL;
      if(i<8){ dead.push([t,t+WAIT]); t+=WAIT; }
    }
    const total=t;
    const el=elastic(total,dead);
    const at=el.at;
    const members=polls.map(([a,b])=>[at(a), at(b)-at(a), 'good']);
    const naps=dead.map(([a,b])=>[at(a), at(b)-at(a), 'waitok']);
    return fig([
      {n:'', segs:[]},
      {n:'', segs:[]},
    ], groupRow(0,{n:'× 9 poll',x:0,w:at(total),members,note:'9 · 0 failed'})+
       groupRow(1,{n:'× 8 nap',x:at(polls[0][1]),w:at(dead[7][1])-at(polls[0][1]),members:naps}),
      'eight compressions sharing one budget','',
      {margin:0,breaks:el.bands.map(b=>[b.p0,b.p1,b.label?'2m':''])});
  })()});

D('t1d',{d:'A wait that has not resolved still compresses. The run is asleep right now: the bar is blue and carries no closing mark, because the missing mark is what says unresolved &mdash; and the dead time inside it is real whether or not it has finished. There is no Finalization row, because the run has not been finalized.',
  svg:(()=>{
    const total=20+3*3600;
    const el=elastic(total, [], {compute:[[0,20]]});
    const g=(k,a,b)=>[k, el.at(a), el.at(b)-el.at(a)];
    const bd=el.bands[0];
    return fig([
      {n:'a',   segs:[g('good',0,20)]},
      {n:'nap', segs:[g('wait',20,total)],note:'sleeping'},
    ],'','an unresolved wait, compressed','',
      {margin:0,running:true,breaks:[[bd.p0,bd.p1,'3h']]});
  })()});

D('t1e',{d:'Choosing where to cut. A gap in <em>one</em> row is not dead time &mdash; something else may be running through it &mdash; so the compute of every row is merged first and only the holes in that union are candidates. Here <code>b</code> works straight through <code>a</code>&rsquo;s wait, so nothing is compressed there however long it looks; the one stretch with nothing running anywhere is.',
  svg:(()=>{
    const compute=[[0,8],[4,26],[900,914],[914,930]];
    const total=940;
    const el=elastic(total, [], {compute});
    const g=(k,a,b)=>[k, el.at(a), el.at(b)-el.at(a)];
    return fig([
      {n:'a', segs:[g('good',0,8),g('waitok',8,26)]},
      {n:'b', segs:[g('good',4,26)]},
      {n:'c', segs:[g('waitok',26,900),g('good',900,914)]},
      {n:'d', segs:[g('good',914,930)]},
    ],'','only the stretch with nothing running','',
      {margin:0,breaks:el.bands.map(b=>[b.p0,b.p1,'15m'])});
  })()});

D('t2',{d:'Seven days elapsed, 62ms executing. Reading the fill alone tells you that before you have read a number, which is the point of the height rule.',
  svg:fig([
    {run:true, end:100, intervals:[{a:0,b:1.2,ok:true},{a:58,b:59,ok:true},{a:63,b:64.5,ok:true}], resolvedAt:100, resolvedAs:EV.ok},
    {n:'a', segs:[['good',0,1.2]]},
    {n:'nap', segs:[['waitok',1.2,56.8]]},
    {n:'b', segs:[['disc',58,1],['idle',59,4],['good',63,1.5]]},
  ],'','62ms of execution inside seven days')});

D('t3',{d:'Two tiers of axis label. The coarse tier carries what the run crossed, the fine tier carries offsets inside it, so a run measured in days keeps its resolution without a second axis.',
  // No rows. This figure is about the axis, so it draws the axis and nothing
  // else — put under a trace it reads as a stray timeline left in the middle of
  // a run, which is exactly what it looked like.
  svg:fig([],axis(cy(0)+2,[[0,'0'],[25,'+6h'],[50,'+12h'],[75,'+18h'],[100,'+24h']],
                 {tier2:[[0,'Mar 4'],[50,'Mar 5']]}),
    'two-tier axis labels','',{frame:false,rowCount:1,pad:26})});

D('t4',{d:'A step too short to draw is still drawn. It gets a minimum width so it can be pointed at, and the number beside it is the real one: the drawing rounds, the reported duration does not.',
  svg:fig([
    {n:'a', segs:[['good',0,0.4]]},
    {n:'nap',  segs:[['waitok',0.4,96]]},
    {n:'b', segs:[['disc',96.4,0.2],['good',96.6,0.4]]},
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
      {n:'req + worker ×12', segs:[['idle',0,2],['disc',2,2]]},
      {n:'', segs:[]},
      {n:'collect', segs:[['disc',80,2],['idle',82,3],['good',85,12]]},
    ], groupRow(1,{n:'× 12 worker',x:5,w:70,members,note:'12 · staggered'}),
      'a wide fan-out, collapsed');
  })()});

D('s3',{d:'A cluster of failures two thirds through a 500-step run is a red smear on the Run row, findable in the first screenful without scrolling or interacting. This is what the failure-wins rule buys: under the old &ldquo;only if every step agrees&rdquo; rule these slices held successes too, so the cluster drew neutral blue and disappeared at exactly the scale it matters.',
  svg:(()=>{
    // Contiguous slices, not one per step: 500 rounded rects with gaps between
    // them beaded into a dotted strip and stopped reading as one profile. The
    // Run row is a profile of the whole run, so it is drawn as one.
    const intervals=[{a:0,b:54,ok:true},{a:54,b:63,ok:false},{a:63,b:86,ok:true}];
    return fig([
      {run:true, to:86, intervals, resolved:EV.failed},
      {n:'act-[0…499]', segs:[['idle',0,2],['good',2,84]]},
    ], tag(px(54),cy(2)+3,'a failure cluster, 500 steps in'),
      'a failure cluster on the Run row', {pad:12});
  })()});

D('s4',{d:'Whatever the step count, the trace draws about forty rows at rest. Everything past that is inside a collapsed group, and every collapsed group can be expanded where you are standing.',
  svg:fig([
    {n:'first', segs:[['good',0,6]]},
    {n:'× 40 batch', segs:[['idle',6,2],['good',8,60]],note:''},
    {n:'  ↳ batch 7', segs:[['idle',20,1],['good',21,6]]},
    {n:'  ↳ batch 8', segs:[['idle',24,1],['good',25,5]]},
    {n:'last', segs:[['disc',70,2],['idle',72,3],['good',75,12]]},
  ],tag(px(69),cy(1)+2.5,'× 40'),'one group expanded in place')});

// ---- Naming & identity --------------------------------------------------

D('n1',{d:'The same step name on two branches. The name is not the identity: rows are distinguished by which request reported them, which the ribbon already shows.',
  svg:fig([
    {n:'req + fetch', segs:[['idle',0,2.4],['disc',2.4,4.6],['good',7,26]]},
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

D('i2',{d:'The Run row <em>is</em> the overview. There was a minimap above it drawing the same run in the same place in the same colours &mdash; two pictures of one fact, and a second overview that could drift out of step with the first. One picture of the run, not two.',
  svg:(()=>{
    // Failure wins over mixed here: the cluster at 48 is what an overview is
    // for, and the old "only if every step agrees" rule drew it neutral blue.
    const intervals=[{a:0,b:18,ok:true},{a:18,b:48,ok:true},
                     {a:48,b:58,ok:false},{a:58,b:86,ok:true}];
    return fig([
      {run:true, to:86, intervals, resolved:EV.ok},
      {n:'b', segs:[['idle',30,4],['good',34,26]]},
      {n:'c', segs:[['bad',60,10]]},
    ], '', 'the Run row is the overview and the scrubber');
  })()});

// ---- OpenTelemetry ------------------------------------------------------

D('o0',{d:'How they get into the trace. At rest a step with userland spans is still one row, with a <strong>coverage rail</strong> under its bar: where spans covered the execution, and whether any of them failed. It never tints the bar &mdash; a step that returned is green whatever happened in a span inside it, because those are different facts. The break in the rail is your code doing something no span covers, which is worth seeing on its own.',
  frames:(()=>{
    const step={n:'charge',segs:[['idle',0,6],['good',6,74]],
      rail:[[9,29,'unset'],[49,12,'unset'],[63,14,'unset']],note:'3 spans'};
    return [
      {l:'at rest', svg:fig([step],'','coverage under the step')},
      {l:'expanded', svg:fig([
        {n:'charge',segs:[['idle',0,6],['good',6,74]]},
        {n:'  ↳ POST /pay', segs:[['spanunset',9,29]],thin:true},
        {n:'  ↳ SELECT',    segs:[['spanunset',49,12]],thin:true},
        {n:'  ↳ UPDATE',    segs:[['spanunset',63,14]],thin:true},
      ],'','the same spans, expanded')},
    ];
  })()});

D('o0b',{d:'Why it is coverage and not one mark per span. Inside a <code>Promise.all</code> the spans overlap, and a notch or a tick per span would either stack into a smudge or imply an order that is not there. The rail reports the <em>union</em>: which parts of the step something was covering, and the worst status in each part. Anything finer is the expanded view&rsquo;s job.',
  frames:(()=>{
    const step={n:'fanout',segs:[['idle',0,5],['good',5,80]],
      rail:[[8,34,'unset'],[42,26,'err'],[68,15,'unset']],note:'9 spans · 1 error'};
    return [
      {l:'at rest', svg:fig([step],'','overlapping spans, one rail')},
      {l:'expanded', svg:fig([
        {n:'fanout',segs:[['idle',0,5],['good',5,80]]},
        {n:'  ↳ GET /a',  segs:[['spanunset',8,30]],thin:true},
        {n:'  ↳ GET /b',  segs:[['spanunset',10,26]],thin:true},
        {n:'  ↳ GET /c',  segs:[['spanerr',12,32]],thin:true,note:'ERROR'},
        {n:'  ↳ SELECT',  segs:[['spanok',46,22]],thin:true},
        {n:'  ↳ UPDATE',  segs:[['spanunset',68,15]],thin:true},
      ],'','five of the nine, expanded')},
    ];
  })()});

D('o1',{d:'A span keeps the three states OpenTelemetry gives it. Most instrumentation sets none, so <strong>Unset is the common case and must not read as an outcome</strong>: grey means it ran and nobody said, green means something said OK, red means it failed. The step above is green regardless &mdash; it returned.',
  svg:fig([
    {n:'charge',       segs:[['idle',0,6],['good',6,74]],rail:[[9,20,'unset'],[31,20,'err'],[53,24,'ok']]},
    {n:'  ↳ POST /pay',segs:[['spanunset',9,20]],thin:true,note:'Unset'},
    {n:'  ↳ SELECT',   segs:[['spanerr',31,20]],thin:true,note:'ERROR'},
    {n:'  ↳ UPDATE',   segs:[['spanok',53,24]],thin:true,note:'OK'},
  ],'','the three span statuses')});

D('o2',{d:'The nesting is a claim, and the trace has to be able to keep it. A span always sits <em>inside</em> its step&rsquo;s execution, because that is where it ran &mdash; one that starts before its step or outruns it is a clock disagreement between your process and ours, not a slow query, and drawing it as though it were would be the view inventing a fact. Where the extents do not contain each other the row says so rather than clamping quietly.',
  svg:fig([
    {n:'charge',        segs:[['idle',0,6],['good',6,54]]},
    {n:'  ↳ POST /pay', segs:[['good',9,38]],thin:true},
    {n:'  ↳ SELECT',    segs:[['good',52,26]],thin:true,note:'outruns its step'},
  ],'','a span that leaves its parent')});

// ---- Honesty ------------------------------------------------------------

D('h1',{d:'Where the trace cannot say which step caused a request, it declines rather than guessing. No cable is drawn and the row says so, which is a smaller cost than a confident wrong line.',
  svg:fig([
    {n:'a', segs:[['good',0,20]],dim:DIM},
    {n:'b', segs:[['good',4,24]],dim:DIM},
    {n:'c', segs:[['idle',28,4],['good',32,24]],sel:true},
  ],tag(px(58),cy(2)+2.5,'no reported parent'),'declining to attribute')});

D('h2',{d:'A row’s label and its drawing have to agree. If the bar is drawn across a wider interval than the step ran for, the number beside it names the interval that was widened, not the one the SDK reported.',
  svg:fig([
    {n:'a', segs:[['idle',0,2],['disc',2,1],['idle',3,8],['good',11,20]]},
  ],tag(px(33),cy(0)+2.5,'3ms reported · 31ms on the row'),'the label reconciles with the drawing')});

D('h3',{d:'Nothing is drawn that cannot be asked what it is. Every interval and every mark on every row decomposes into named parts on hover, which is also how this document proves it has no unexplained pixels.',
  svg:fig([
    {n:'req + a', segs:[['idle',0,2.8],['disc',2.8,5.2],['idle',8,4],['good',12,30]]},
  ],'','everything decomposes')});

fs.writeFileSync(HERE+'items-more.json',JSON.stringify(
  Object.fromEntries(Object.entries(E).map(([k,v])=>
    [k,{d:v.d,frames:v.frames||[{l:'at rest',svg:v.svg}]}]))));
console.log('more',Object.keys(E).length);
