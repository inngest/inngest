const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,lineage,px,cy,arrow,dot,C,W,LBL,PLOT,ROW,TOP,setFrame} from './micro.mjs';
import {setFrameSharp} from './micro.mjs'; setFrameSharp(true);
setFrame(true);

/**
 * The captured fixtures went through a SECOND RENDERER — its own `panel`,
 * `row` and `runRow`, its own width, row pitch and label gutter — so they got
 * none of the rules: no frame, no compression, no live geometry, and a Run row
 * they declared for themselves. They looked authored because they were drawn by
 * a different program.
 *
 * This adapts the captured shape onto the one renderer. The Run row and the
 * Finalization row are dropped on the way in, because the frame derives both
 * from the rows beneath — a captured fixture declaring either is a second
 * account of the same run.
 */
const panel=(rows,extra='',label='',ms)=>fig(
  rows.filter(r=>r.run===undefined && r.name!=='Finalization')
      .map(r=>({...r, n:r.name, segs:(r.segs||[]).map(g=>[g.kind,g.x,g.w]),
                note:[r.dur,r.note].filter(Boolean).join(' · ')})),
  // A captured run starts when it starts working; the queue it opened with is
  // named on the Run row instead of pushing every row to the right of it.
  extra, label, '', {trimLead:true, ms});
const F={};

// ---- simple: no steps at all -------------------------------------------
F.simple=[{
  cap:'Two rows and nothing to relate. No ribbon, no cables. The design has to be quiet here or it is noise everywhere else.',
  svg:panel([
    {run:[],lbl:'no step spans · 8ms'},
    {name:'Finalization',dur:'8ms',at:[['started',0],['ok',99.5]]},
  ],'','simple: two rows, no relationships',8),
}];

// ---- step: the baseline shape ------------------------------------------
F.step=[{
  cap:'The sleep is the row. Waiting is its whole substance, so it draws one bar with a circle at each end and no lead-in. The two 12ms steps either side are 2px, honestly, because that is the proportion.',
  svg:panel([
    {run:[{a:0,b:0.5,ok:true},{a:96.9,b:97.4,ok:true}],lbl:'22ms compute / 2.256s'},
    {name:'first step',dur:'12ms',at:[['started',0],['ok',0.5]]},
    {name:'for 2s',dur:'2.001s',note:'+1ms planning',at:[['started',0.5],['ok',96.5]],kind:'wait'},
    {name:'second step',dur:'10ms',note:'+2ms wait',at:[['queued',96.5],['started',96.9],['ok',97.4]]},
    {name:'Finalization',dur:'60ms',note:'+51ms wait',at:[['queued',97.4],['started',99.1],['ok',99.7]]},
  ],'','step: sleep dominates the axis',2256),
}];

// ---- v4sequential: must NOT show parallelism ---------------------------
F.v4sequential=[{
  cap:'The point of this fixture is that nothing here is parallel, on a client that could report batches. Every request produced exactly one step, so <strong>no ribbon anywhere</strong> is the correct render, and it is visibly different from <code>v4parallel</code> without reading a label.',
  svg:panel([
    {run:[{a:0.8,b:1.2,ok:true},{a:99.3,b:99.7,ok:true}],lbl:'2ms compute / 2.050s'},
    {name:'first step',dur:'1ms',at:[['started',0.8],['ok',1.2]]},
    {name:'for 2s',dur:'2.001s',note:'+1ms planning',at:[['started',1.2],['ok',98.2]],kind:'wait'},
    {name:'second step',dur:'1ms',at:[['started',99.3],['ok',99.7]]},
    {name:'Finalization',dur:'5ms',at:[['started',99.7],['ok',100]]},
  ],'','v4sequential: no ribbon anywhere',2050),
}];

// ---- emit: lineage the platform does not label -------------------------
F.emit=[{
  cap:'<code>fan-out</code> sent two events that each started a run. That is real lineage, and it leaves this run entirely, so it is <em>not</em> a ribbon, which groups steps <em>within</em> this trace. A distinct outbound marker keeps the two kinds from being confused.',
  svg:(()=>{
    const rows=[
      {run:[{a:27.3,b:29.8,ok:true},{a:45.5,b:60.1,ok:true},{a:75.8,b:78.3,ok:true}],lbl:'7ms compute / 33ms'},
      {name:'prepare',dur:'1ms',at:[['started',27.3],['ok',29.8]]},
      {name:'fan-out',dur:'5ms',at:[['started',45.5],['ok',60.1]],lineage:2},
      {name:'after',dur:'1ms',at:[['started',75.8],['ok',78.3]]},
      {name:'Finalization',dur:'7ms',at:[['started',78.8],['ok',99.5]]},
    ];
    return panel(rows,'','emit: outbound lineage marker',33);
  })(),
}];

// ---- invoke: a whole other run inside a row -----------------------------
F.invoke=[
 {
  cap:'Today <code>call child</code> draws 17% of its own extent &mdash; the 545ms invocation has no mark at all, leaving the widest void in the gallery. Drawn correctly, the child run is a distinct substance: not your code, not Inngest&rsquo;s, so neither green nor blue.',
  svg:panel([
    {run:[{a:0,b:0.5,ok:true},{a:82.7,b:83.4,ok:true}],lbl:'67ms own compute / 909ms'},
    {name:'before',dur:'9ms',at:[['started',0],['ok',0.5]]},
    {name:'call child',dur:'545ms',note:'+146ms queued · +8ms planning',
     at:[['queued',1],['started',16.5,'disc'],['ok',17.1],['started',17.1],['ok',78.1]],kind:'child',reported:1,
    },
    {name:'after',dur:'58ms',note:'+47ms wait',at:[['queued',78.1],['started',82.7],['ok',83.4]]},
    {name:'Finalization',dur:'141ms',note:'+130ms wait',at:[['queued',84.5],['started',98.3],['ok',99]]},
  ],'','invoke: the child run drawn as its own substance',909),
 },
 {
  cap:'Expanded, the child&rsquo;s own rows indent under it on the same axis. The parent row keeps its circles; the child&rsquo;s rows get theirs, so the boundary between the two runs is legible without a label.',
  svg:panel([
    {name:'call child',dur:'545ms',at:[['queued',1],['started',16.5,'disc'],['ok',17.1],['started',17.1],['ok',78.1]],kind:'child',reported:1},
    {name:'  ↳ Run',dur:'271ms',at:[['started',19],['ok',69]]},
    {name:'  ↳ work',dur:'180ms',at:[['queued',22],['started',30],['ok',63]]},
    {name:'  ↳ finish',dur:'40ms',at:[['queued',63],['started',67],['ok',69]]},
    {name:'after',dur:'58ms',at:[['queued',78.1],['started',82.7],['ok',83.4]]},
  ],'','invoke expanded: child rows on the same axis',909),
 },
];

fs.writeFileSync(HERE+'batch1.json',JSON.stringify(F));
console.log('ok', Object.keys(F).join(' '));
