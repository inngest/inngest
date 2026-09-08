const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,fin,px,cy,tag,axis,groupRow,arrow,dot,elastic,layout,EV,C,W,LBL,PLOT,ROW,TOP} from './micro.mjs';
import {setFrame} from './micro.mjs';
import * as R from './rules.mjs'; setFrame(true);
const E={}; const D=(k,v)=>{E[k]=v;};
const DIM=.22;

// ---- Waiting ------------------------------------------------------------

D('w1',{d:'A wait that matched. The bar is blue while it is open because nothing has decided it, and turns green at the mark where the event arrived.',
  svg:fig([
    {n:'a',              at:[['started',0],['ok',100]]},
    {n:'w',              at:[['started',100],['ok',120],['started',120],['ok',640]],kind:'wait',reported:1},
    {n:'b',              at:[['started',640],['ok',660],['queued',660],['started',700],['ok',860]],reported:1},
    fin(860,45,40),
  ],'','a wait that matched')});

D('w2',{d:'A wait that expired without a match. The run carries on: the timeout is a result the function can act on, so the row goes grey and the next step is reported as normal.',
  svg:fig([
    {n:'a',              at:[['started',0],['ok',100]]},
    {n:'w',              at:[['started',100],['ok',120],['started',120],['timeout',640]],kind:'wait',reported:1},
    {n:'fallback',       at:[['started',640],['ok',660],['queued',660],['started',700],['ok',860]],reported:1},
    fin(860,45,40),
  ],'','a wait that timed out, run continues')});

D('w3',{d:'A wait inside a fan-out. It sits on the same axis as its siblings and holds the level open: the request that collects them cannot start until the slowest resolves, which the queue time on the next row reports.',
  svg:fig([
    {n:'req + a', at:[['queued',0],['started',28,'disc'],['ok',80],['started',80],['ok',280]],reported:1},
    {n:'b',       at:[['started',80],['ok',220]]},
    {n:'c',       at:[['started',80],['ok',620]],kind:'wait'},
    {n:'d',       at:[['started',620],['ok',640],['queued',640],['started',680],['ok',860]],reported:1},
    fin(860,45,40),
  ],'','a wait holding a level open')});

D('w4',{d:'Waiting and failing must not read alike. One is hatched and never red; the other is solid and red, and only its final attempt takes a filled mark.',
  svg:fig([
    {n:'nap',      at:[['started',0]],kind:'wait',end:600},
    {n:'a',        at:[['started',0],['retry',260],['queued',400],['started',440],['failed',700]]},
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
    const _rows=[
      {n:'a',   at:[['started',0],['ok',41]]},
      {n:'nap', at:[['started',41],['ok',41+NAP]],kind:'wait',note:'7d'},
      {n:'b',   at:[['queued',B0],['started',B0+6,'disc'],['ok',B0+16],['queued',B0+16],['started',B0+22],['ok',total]],reported:1},
      fin(total,8,12),
    ];
    return fig(_rows,'','seven days compressed to a band','',{margin:0,ms:total});
  })()});

D('t1b',{d:'What the space left over is worth. Thirty seconds of work on one side of a compressed gap and ten on the other, so the remaining width splits 75/25 &mdash; which is the same thing as saying <strong>a second is the same number of pixels wherever it lands</strong>. Without that rule, compressing a gap would quietly rescale one half of the trace against the other and two spans either side of it would stop being comparable. It is also why several compressions need no special case: the arithmetic is total live width over total live time, applied everywhere.',
  svg:(()=>{
    const GAP=7*86400, total=30+GAP+10;
    const _rows=[
      {n:'left',  at:[['started',0],['ok',30]],note:'30s'},
      {n:'gap',   at:[['started',30],['ok',30+GAP]],kind:'wait'},
      {n:'right', at:[['started',30+GAP],['ok',total]],note:'10s'},
      fin(total,0.2,0.3),
    ];
    return fig(_rows,'','the leftover width splits by real duration','',{margin:0,ms:total,unit:'s'});
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
    return fig([
      {n:'', group:{n:'× 9 poll', to:total, note:'9 · 0 failed',
                    members:polls.map(([a,b])=>[a,b,'good'])}},
      {n:'', group:{n:'× 8 nap', to:total,
                    members:dead.map(([a,b])=>[a,b,'waitok'])}},
      fin(total,2,3),
    ],'','eight compressions sharing one budget','',{margin:0, ms:total, unit:'s'});
  })()});

D('t1d',{d:'A wait that has not resolved still compresses. The run is asleep right now: the bar is blue and carries no closing mark, because the missing mark is what says unresolved &mdash; and the dead time inside it is real whether or not it has finished. There is no Finalization row, because the run has not been finalized.',
  svg:(()=>{
    const total=0.34+7*86400;
    const _rows=[
      {n:'a',   at:[['started',0],['ok',0.34]]},
      {n:'nap', at:[['started',0.34]],end:total,kind:'wait',note:'sleeping'},
    ];
    return fig(_rows,'','an unresolved wait, compressed','',
      {margin:0,running:true,ms:total,unit:'s'});
  })()});

D('t1e',{d:'Choosing where to cut. A gap in <em>one</em> row is not dead time &mdash; something else may be running through it &mdash; so the compute of every row is merged first and only the holes in that union are candidates. Here <code>b</code> works straight through <code>a</code>&rsquo;s wait, so nothing is compressed there however long it looks; the one stretch with nothing running anywhere is.',
  svg:(()=>{
    const total=940;
    const _rows=[
      {n:'a', at:[['started',0],['ok',8],['started',8,'wait'],['ok',26]]},
      {n:'b', at:[['started',4],['ok',26]]},
      {n:'c', at:[['started',26],['ok',900],['started',900],['ok',914]],kind:'wait'},
      {n:'d', at:[['started',914],['ok',930]]},
      fin(930,3,5),
    ];
    return fig(_rows,'','only the stretch with nothing running','',{margin:0,ms:total,unit:'s'});
  })()});

D('t2',{d:'Seven days elapsed, 62ms executing. Reading the fill alone tells you that before you have read a number, which is the point of the height rule.',
  // Real durations, and no Run row of its own: the row this figure is ARGUING
  // for was written out here by hand, intervals and all, which is the one
  // thing the argument is against. It is derived from the rows now, like
  // everywhere else.
  svg:(()=>{
    const B=12+NAP;
    return fig([
      {n:'a',   at:[['started',0],['ok',12]]},
      {n:'nap', at:[['started',12],['ok',B]],kind:'wait'},
      {n:'b',   at:[['queued',B],['started',B+10,'disc'],['ok',B+20],['queued',B+20],['started',B+20],['ok',B+50]],reported:1},
      fin(B+50,8,10),
    ],'','62ms of execution inside seven days','');
  })()});

D('t3',{d:'Two tiers of axis label. The coarse tier carries what the run crossed, the fine tier carries offsets inside it, so a run measured in days keeps its resolution without a second axis.',
  // No rows. This figure is about the axis, so it draws the axis and nothing
  // else — put under a trace it reads as a stray timeline left in the middle of
  // a run, which is exactly what it looked like.
  svg:fig([],axis(cy(0)+2,[[0,'0'],[25,'+6h'],[50,'+12h'],[75,'+18h'],[100,'+24h']],
                 {tier2:[[0,'Mar 4'],[50,'Mar 5']]}),
    'two-tier axis labels','',{frame:false,rowCount:1,pad:26})});

D('t4',{d:'A step too short to draw is still drawn. It gets a minimum width so it can be pointed at, and the number beside it is the real one: the drawing rounds, the reported duration does not.',
  svg:fig([
    {n:'a', at:[['started',0],['ok',12]],note:'12ms'},
    {n:'nap',  at:[['started',12],['ok',7*864e5+12]],kind:'wait'},
    {n:'b', at:[['started',7*864e5+12],['ok',7*864e5+22],['started',7*864e5+22],['ok',7*864e5+32]],reported:1,note:'10ms'},
    fin(7*864e5+32,8,12),
  ],'','sub-pixel steps drawn honestly','')});

// ---- Scale --------------------------------------------------------------

D('s1',{d:'Five hundred sequential steps do not become five hundred rows. Repetition with one shape collapses to a single row that reports the count and draws where each member ran.',
  svg:(()=>{
    const members=[]; for(let i=0;i<36;i++) members.push([4+i*2.1, 1.3, 'good']);
    return fig([
      {n:'setup', at:[['started',0],['ok',30]]},
      {n:''},
      {n:'teardown', at:[['queued',880],['started',900],['ok',980]]},
      fin(980,40,50),
    ], groupRow(1,{n:'× 500 fetch',x:3,w:79,members,note:'500 · 0 failed'}),
      'five hundred steps, one row','',{busy:[[3,82]]});
  })()});

D('s2',{d:'The same collapse for a wide fan-out. The envelope is the level and the ticks are the members, so the stagger of a concurrency limit is visible without expanding anything.',
  svg:(()=>{
    const members=[]; for(let i=0;i<12;i++) members.push([6+i*2.9, 21-i*0.6, 'good']);
    return fig([
      {n:'req + worker ×12', at:[['queued',0],['started',20],['ok',40]],kind:'disc'},
      {n:''},
      {n:'collect', at:[['started',800],['ok',820],['queued',820],['started',850],['ok',970]],reported:1},
      fin(970,40,50),
    ], groupRow(1,{n:'× 12 worker',x:5,w:70,members,note:'12 · staggered'}),
      'a wide fan-out, collapsed','',{busy:[[5,75]]});
  })()});

D('s3',{d:'A cluster of failures two thirds through a 500-step run is a red smear on the Run row, findable in the first screenful without scrolling or interacting. This is what the failure-wins rule buys: under the old &ldquo;only if every step agrees&rdquo; rule these slices held successes too, so the cluster drew neutral blue and disappeared at exactly the scale it matters.',
  svg:(()=>{
    // Contiguous slices, not one per step: 500 rounded rects with gaps between
    // them beaded into a dotted strip and stopped reading as one profile. The
    // Run row is a profile of the whole run, so it is drawn as one.
    const intervals=[{a:0,b:54,ok:true},{a:54,b:63,ok:false},{a:63,b:86,ok:true}];
    return fig([
      {run:true, to:86, intervals, resolved:EV.failed},
      {n:'act-[0…499]', at:[['queued',0],['started',20],['ok',860]]},
    ], '',
      'a failure cluster on the Run row', {pad:12});
  })()});

D('s4',{d:'Whatever the step count, the trace draws about forty rows at rest. Everything past that is inside a collapsed group, and every collapsed group can be expanded where you are standing.',
  svg:fig([
    {n:'first', at:[['started',0],['ok',60]]},
    {n:'× 40 batch', at:[['queued',60],['started',80],['ok',680]],note:''},
    {n:'  ↳ batch 7', at:[['queued',200],['started',210],['ok',270]]},
    {n:'  ↳ batch 8', at:[['queued',240],['started',250],['ok',300]]},
    {n:'last', at:[['started',700],['ok',720],['queued',720],['started',750],['ok',870]],reported:1},
    fin(870,40,50),
  ],'','one group expanded in place')});

// ---- Naming & identity --------------------------------------------------

D('n1',{d:'The same step name on two branches. The name is not the identity: rows are distinguished by which request reported them, which the ribbon already shows.',
  svg:fig([
    {n:'req + fetch', at:[['queued',0],['started',24,'disc'],['ok',70],['started',70],['ok',330]],reported:1},
    {n:'fetch',       at:[['started',70],['ok',250]]},
    {n:'save',        at:[['started',330],['ok',350],['queued',350],['started',380],['ok',580]],reported:1},
    fin(580,30,40),
  ],'','the same name twice')});

D('n2',{d:'A step id used twice is two steps, and the run numbers the repeats so the rows can be told apart. The number does that and nothing else: it is not stable under parallelism, so no relationship is read from it — which of these came first, or which branch either belongs to, comes from the request that reported them.',
  svg:fig([
    {n:'fetch:1', at:[['started',60],['ok',260]]},
    {n:'fetch:2', at:[['started',60],['ok',360]]},
    fin(360,25,30),
  ],'','unstable suffixes')});

// ---- Interaction --------------------------------------------------------

D('i1',{d:'Three tiers of attention on hover: the row itself, what caused it, and everything else. Nothing is removed, only quietened, so the shape of the run is still readable while you read one row.',
  svg:fig([
    {n:'a',  at:[['started',0],['ok',180]],dim:1},
    {n:'b',  at:[['queued',180],['started',220],['ok',480]],sel:true},
    {n:'c',  at:[['started',40],['ok',160]],dim:DIM},
    {n:'d',  at:[['queued',480],['started',510],['ok',710]],dim:DIM},
    {...fin(710,35,40),dim:DIM},
  ],'','hovered, immediate cause, everything else')});

D('i2',{d:'The Run row <em>is</em> the overview. There was a minimap above it drawing the same run in the same place in the same colours &mdash; two pictures of one fact, and a second overview that could drift out of step with the first. One picture of the run, not two.',
  svg:(()=>{
    // Failure wins over mixed here: the cluster at 48 is what an overview is
    // for, and the old "only if every step agrees" rule drew it neutral blue.
    const intervals=[{a:0,b:18,ok:true},{a:18,b:48,ok:true},
                     {a:48,b:58,ok:false},{a:58,b:86,ok:true}];
    return fig([
      {run:true, to:86, intervals, resolved:EV.ok},
      {n:'b', at:[['queued',300],['started',340],['ok',600]]},
      {n:'c', at:[['started',600],['failed',700]]},
    ], '', 'the Run row is the overview');
  })()});

// ---- OpenTelemetry ------------------------------------------------------

D('o0',{d:'A step with userland spans says so in the gutter and nothing else. Selecting it expands them. They are drawn as <strong>plain bars on a tighter pitch</strong> &mdash; no queue marks, no start or resolution circles &mdash; because the event vocabulary belongs to the trace, and a span is something that happened inside one row of it. Packed until they nearly touch, they read as a block belonging to the step above rather than as more of the run.',
  frames:(()=>{
    const step={n:'charge',at:[['queued',0],['started',60],['ok',800]],spans:true};
    return [
      {l:'at rest', svg:fig([step,fin(800,40,50)],'','the gutter says there is more')},
      {l:'selected', svg:fig([
        {...step,sel:true},
        {n:'POST /pay', at:[['started',90],['done',380]],kind:'span',span:true},
        {n:'SELECT',    at:[['started',490],['done',610]],kind:'span',span:true},
        {n:'UPDATE',    at:[['started',630],['done',770]],kind:'span',span:true},
        fin(800,40,50),
      ],'','expanded on select')},
    ];
  })()});

D('o0b',{d:'The same at any complexity. Spans inside a <code>Promise.all</code> overlap, and on their own pitch that reads as exactly what it is &mdash; several things running at once inside one step &mdash; without a ribbon, a mark or an ordering claim, none of which apply to something the trace did not schedule.',
  frames:(()=>{
    const step={n:'fanout',at:[['queued',0],['started',50],['ok',850]],spans:true};
    return [
      {l:'at rest', svg:fig([step,fin(850,40,50)],'','nine spans, one row')},
      {l:'selected', svg:fig([
        {...step,sel:true},
        {n:'GET /a', at:[['started',80],['done',380]],kind:'span',span:true},
        {n:'GET /b', at:[['started',100],['done',360]],kind:'span',span:true},
        {n:'GET /c', at:[['started',120],['failed',440]],kind:'span',span:true},
        {n:'SELECT', at:[['started',460],['ok',680]],kind:'span',span:true},
        {n:'UPDATE', at:[['started',680],['done',830]],kind:'span',span:true},
        fin(850,40,50),
      ],'','overlapping spans on their own pitch')},
    ];
  })()});

D('o1',{d:'A span keeps the three states OpenTelemetry gives it. Most instrumentation sets none, so <strong>Unset is the common case and must not read as an outcome</strong>: grey means it ran and nobody said, green means something said OK, red means it failed. <code>charge</code> is green regardless &mdash; it returned, and what happened in a span inside it is a different fact.',
  svg:fig([
    {n:'charge',   at:[['queued',0],['started',60],['ok',800]],spans:true},
    {n:'POST /pay',at:[['started',90],['done',290]],kind:'span',span:true,note:'Unset'},
    {n:'SELECT',   at:[['started',310],['failed',510]],kind:'span',span:true,note:'ERROR'},
    {n:'UPDATE',   at:[['started',530],['ok',770]],kind:'span',span:true,note:'OK'},
    fin(800,40,50),
  ],'','the three span statuses')});

D('o2',{d:'The nesting is a claim, and the trace has to be able to keep it. A span always sits <em>inside</em> its step&rsquo;s execution, because that is where it ran &mdash; one that starts before its step or outruns it is a clock disagreement between your process and ours, not a slow query, and drawing it as though it were would be the view inventing a fact. Where the extents do not contain each other the row says so rather than clamping quietly.',
  svg:fig([
    {n:'charge',   at:[['queued',0],['started',60],['ok',600]],spans:true},
    {n:'POST /pay',at:[['started',90],['done',470]],kind:'span',span:true},
    {n:'SELECT',   at:[['started',520],['done',780]],kind:'span',span:true,note:'outruns its step'},
    fin(780,35,45),
  ],'','a span that leaves its parent')});

// ---- Honesty ------------------------------------------------------------

D('h1',{d:'Where the trace cannot say which step caused a request, it declines rather than guessing. No cable is drawn and the row says so, which is a smaller cost than a confident wrong line.',
  svg:fig([
    {n:'a', at:[['started',0],['ok',200]],dim:DIM},
    {n:'b', at:[['started',40],['ok',280]],dim:DIM},
    {n:'c', at:[['queued',280],['started',320],['ok',560]],sel:true},
    fin(560,30,35),
  ],'','declining to attribute','',{attributed:false})});

D('h2',{d:'A row’s label and its drawing have to agree. If the bar is drawn across a wider interval than the step ran for, the number beside it names the interval that was widened, not the one the SDK reported.',
  svg:fig([
    {n:'a', at:[['queued',0],['started',20,'disc'],['ok',30],['queued',30],['started',110],['ok',310]],reported:1,
     note:'3ms reported · 31ms on the row'},
    fin(310,25,30),
  ],'','the label reconciles with the drawing')});


fs.writeFileSync(HERE+'items-more.json',JSON.stringify(
  Object.fromEntries(Object.entries(E).map(([k,v])=>
    [k,{d:v.d,frames:v.frames||[{l:'at rest',svg:v.svg}]}]))));
console.log('more',Object.keys(E).length);
