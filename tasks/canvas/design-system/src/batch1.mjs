const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,lineage,px,cy,arrow,dot,C,W,LBL,PLOT,ROW,TOP,setFrame,layout} from './micro.mjs';
import {loadRun} from './runs.mjs';
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
 * row is dropped on the way in, because the frame derives it. Finalization is
 * NOT dropped: a captured run measured its own, and the frame's invented one
 * sat at a fixed 90% with bars it made up.
 * from the rows beneath — a captured fixture declaring either is a second
 * account of the same run.
 */
/**
 * A captured run, drawn.
 *
 * `loadRun` reads the payload the API returns and gives back the events. The
 * proportions used to be typed out here by hand from those same payloads --
 * a second copy of a measurement, which drifted from it: the transcriptions had
 * no opening queue at all, and gave `simple` an 8ms run that really took 67.
 */
const capture=(id,extra='',label='')=>{
  const r=loadRun(id);
  // The events, and how long the run took. fig() owns everything after that --
  // trimming the opening queue, the elastic axis, the bands -- so the same
  // events redraw differently when what a drawing means changes.
  return fig(r.rows, extra, label, '', {ms:r.ms, trimLead:true});
};

const panel=(rows,extra='',label='',ms)=>fig(
  rows.filter(r=>r.run===undefined)
      .map(r=>({...r, n:r.name, segs:(r.segs||[]).map(g=>[g.kind,g.x,g.w]),
                note:[r.dur,r.note].filter(Boolean).join(' · ')})),
  // A captured run starts when it starts working; the queue it opened with is
  // named on the Run row instead of pushing every row to the right of it.
  extra, label, '', {trimLead:true, ms});
const F={};

// ---- simple: no steps at all -------------------------------------------
F.simple=[{
  cap:'Two rows and nothing to relate. No ribbon, no cables. The design has to be quiet here or it is noise everywhere else.',
  svg:capture('simple','','simple: two rows, no relationships'),
}];

// ---- step: the baseline shape ------------------------------------------
F.step=[{
  cap:'The sleep is the row. Waiting is its whole substance, so it draws one bar with a circle at each end and no lead-in. The two 12ms steps either side are 2px, honestly, because that is the proportion.',
  svg:capture('step','','step: sleep dominates the axis'),
}];

// ---- v4sequential: must NOT show parallelism ---------------------------
F.v4sequential=[{
  cap:'The point of this fixture is that nothing here is parallel, on a client that could report batches. Every request produced exactly one step, so <strong>no ribbon anywhere</strong> is the correct render, and it is visibly different from <code>v4parallel</code> without reading a label.',
  svg:capture('v4sequential','','v4sequential: no ribbon anywhere'),
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
  svg:capture('invoke','','invoke: the child run drawn as its own substance'),
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
