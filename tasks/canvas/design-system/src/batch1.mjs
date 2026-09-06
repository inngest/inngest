const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {lineage,panel,px,cy,arrow,dot,C,W,LBL,PLOT,ROW,TOP,runRow} from './render.mjs';
const g=JSON.parse(fs.readFileSync(HERE+'geom.json','utf8'));
const F={};
const note=(t,i,x=LBL)=>`<text x="${x}" y="${cy(i)+4}" font-family="JetBrains Mono, monospace" font-size="7.5" fill="${C.mut}">${t}</text>`;

// ---- simple: no steps at all -------------------------------------------
F.simple=[{
  cap:'Two rows and nothing to relate. No ribbon, no cables. The design has to be quiet here or it is noise everywhere else.',
  svg:panel([
    {run:[],lbl:'no step spans · 8ms'},
    {name:'Finalization',dur:'8ms',segs:[{x:0,w:99.5,kind:'good'}]},
  ],'','simple: two rows, no relationships'),
}];

// ---- step: the baseline shape ------------------------------------------
F.step=[{
  cap:'The sleep is the row. Waiting is its whole substance, so it draws one bar with a circle at each end and no lead-in. The two 12ms steps either side are 2px, honestly, because that is the proportion.',
  svg:panel([
    {run:[{a:0,b:0.5,ok:true},{a:96.9,b:97.4,ok:true}],lbl:'22ms compute / 2.256s'},
    {name:'first step',dur:'12ms',segs:[{x:0,w:0.5,kind:'good'},]},
    {name:'for 2s',dur:'2.001s',note:'+1ms planning',segs:[{x:0.5,w:96,kind:'waitok'}]},
    {name:'second step',dur:'10ms',note:'+2ms wait',segs:[{x:96.5,w:0.4,kind:'idle'},{x:96.9,w:0.5,kind:'good'}]},
    {name:'Finalization',dur:'60ms',note:'+51ms wait',segs:[{x:97.4,w:1.7,kind:'idle'},{x:99.1,w:0.6,kind:'good'}]},
  ],'','step: sleep dominates the axis'),
}];

// ---- v4sequential: must NOT show parallelism ---------------------------
F.v4sequential=[{
  cap:'The point of this fixture is that nothing here is parallel, on a client that could report batches. Every request produced exactly one step, so <strong>no ribbon anywhere</strong> is the correct render, and it is visibly different from <code>v4parallel</code> without reading a label.',
  svg:panel([
    {run:[{a:0.8,b:1.2,ok:true},{a:99.3,b:99.7,ok:true}],lbl:'2ms compute / 2.050s'},
    {name:'first step',dur:'1ms',segs:[{x:0.8,w:0.4,kind:'good'}]},
    {name:'for 2s',dur:'2.001s',note:'+1ms planning',segs:[{x:1.2,w:97,kind:'waitok'}]},
    {name:'second step',dur:'1ms',segs:[{x:99.3,w:0.4,kind:'good'}]},
    {name:'Finalization',dur:'5ms',segs:[{x:99.7,w:0.3,kind:'good'}]},
  ],'','v4sequential: no ribbon anywhere'),
}];

// ---- emit: lineage the platform does not label -------------------------
F.emit=[{
  cap:'<code>fan-out</code> sent two events that each started a run. That is real lineage, and it leaves this run entirely, so it is <em>not</em> a ribbon, which groups steps <em>within</em> this trace. A distinct outbound marker keeps the two kinds from being confused.',
  svg:(()=>{
    const rows=[
      {run:[{a:27.3,b:29.8,ok:true},{a:45.5,b:60.1,ok:true},{a:75.8,b:78.3,ok:true}],lbl:'7ms compute / 33ms'},
      {name:'prepare',dur:'1ms',segs:[{x:27.3,w:2.5,kind:'good'}]},
      {name:'fan-out',dur:'5ms',segs:[{x:45.5,w:14.6,kind:'good'}]},
      {name:'after',dur:'1ms',segs:[{x:75.8,w:2.5,kind:'good'}]},
      {name:'Finalization',dur:'7ms',segs:[{x:78.8,w:20.7,kind:'good'}]},
    ];
    const y=cy(2), x=px(60.1);
    const out=lineage(x,y,2);
    return panel(rows,out,'emit: outbound lineage marker');
  })(),
}];

// ---- invoke: a whole other run inside a row -----------------------------
F.invoke=[
 {
  cap:'Today <code>call child</code> draws 17% of its own extent &mdash; the 545ms invocation has no mark at all, leaving the widest void in the gallery. Drawn correctly, the child run is a distinct substance: not your code, not Inngest&rsquo;s, so neither green nor blue.',
  svg:panel([
    {run:[{a:0,b:0.5,ok:true},{a:82.7,b:83.4,ok:true}],lbl:'67ms own compute / 909ms'},
    {name:'before',dur:'9ms',segs:[{x:0,w:0.5,kind:'good'},]},
    {name:'call child',dur:'545ms',note:'+146ms queued · +8ms planning',
     segs:[{x:1,w:15.5,kind:'idle'},{x:16.5,w:0.6,kind:'disc'},{x:17.1,w:61,kind:'child'}],
    },
    {name:'after',dur:'58ms',note:'+47ms wait',segs:[{x:78.1,w:4.6,kind:'idle'},{x:82.7,w:0.7,kind:'good'}]},
    {name:'Finalization',dur:'141ms',note:'+130ms wait',segs:[{x:84.5,w:13.8,kind:'idle'},{x:98.3,w:0.7,kind:'good'}]},
  ],'','invoke: the child run drawn as its own substance'),
 },
 {
  cap:'Expanded, the child&rsquo;s own rows indent under it on the same axis. The parent row keeps its circles; the child&rsquo;s rows get theirs, so the boundary between the two runs is legible without a label.',
  svg:panel([
    {name:'call child',dur:'545ms',segs:[{x:1,w:15.5,kind:'idle'},{x:16.5,w:0.6,kind:'disc'},{x:17.1,w:61,kind:'child'}]},
    {name:'  ↳ Run',dur:'271ms',segs:[{x:19,w:50,kind:'good'}]},
    {name:'  ↳ work',dur:'180ms',segs:[{x:22,w:8,kind:'idle'},{x:30,w:33,kind:'good'}]},
    {name:'  ↳ finish',dur:'40ms',segs:[{x:63,w:4,kind:'idle'},{x:67,w:2,kind:'good'}]},
    {name:'after',dur:'58ms',segs:[{x:78.1,w:4.6,kind:'idle'},{x:82.7,w:0.7,kind:'good'}]},
  ],'','invoke expanded: child rows on the same axis'),
 },
];

fs.writeFileSync(HERE+'batch1.json',JSON.stringify(F));
console.log('ok', Object.keys(F).join(' '));
