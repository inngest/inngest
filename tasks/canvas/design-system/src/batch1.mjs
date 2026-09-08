const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,setFrame} from './micro.mjs';
import {loadRun} from './runs.mjs';
import {fixtureCode} from './fixcode.mjs';
setFrame(true);

/**
 * A captured run, drawn.
 *
 * `loadRun` reads the payload the API returns and gives back the events. The
 * proportions used to be typed out here by hand from those same payloads -- a
 * second copy of a measurement, which drifted from it: the transcriptions had
 * no opening queue at all, and gave `simple` an 8ms run that really took 67.
 * There is no hand-drawn fixture left; `panel()`, the last of them, went when
 * every shape became a capture.
 */
const capture=(id,label='')=>{
  const r=loadRun(id);
  // The events, and how long the run took. fig() owns everything after that --
  // trimming the opening queue, the elastic axis, the bands -- so the same
  // events redraw differently when what a drawing means changes.
  return fig(r.rows, '', label, '', {ms:r.ms, trimLead:true});
};

/**
 * Every shape, captured BOTH WAYS.
 *
 * The same function, the same event, two apps that differ in one setting:
 * whether the SDK checkpoints a step it can run inline. With checkpointing the
 * step is reported out of band while the request is still open, so the SDK
 * carries on and one request covers what would otherwise take several -- `emit`
 * is one request instead of four, `sequential` two instead of four. Anything
 * that differs between a pair is the checkpointing and nothing else, which is
 * what makes them worth putting side by side.
 */
const PAIRS=[
  ['simple','No steps at all. The whole run is the request that asks what to do and is told nothing.'],
  ['sequential','A step, a sleep, a step. The sleep cannot be run inline either way, so this is where checkpointing shows most plainly.'],
  ['emit','Three steps in a row with nothing between them, so every one of them can be run inline.'],
  ['loop','Twelve steps of one shape. With checkpointing the whole loop is a single request.'],
  ['parallel','A parallel batch has to be planned, so the fan-out is the same either way and only the step after it moves.'],
  ['wide','Twelve steps planned by one request, then a step that waits for all of them.'],
  ['chains','Parallel branches that are chains: planning and checkpointing in the same run.'],
  ['nested','A fan-out whose branches are themselves fan-outs.'],
  ['nested-in-branch','A branch that fans out: one resumption discovers two steps.'],
  ['unbalanced','Branches of different depths, so the levels stop lining up.'],
  ['dynamic','Fan-out width decided by a previous step, so nothing static predicted the shape.'],
  ['dupe-names','The same step name in both branches. Nothing in the drawing may depend on the suffix.'],
  ['deep3','Three chains, three deep, jittered so completion order varies between runs.'],
  ['race','Promise.race: every branch schedules its own discovery, because racing skips coalescing.'],
  ['mixed','A failing branch beside a succeeding one, caught. One red row must not colour the level.'],
  ['dead-end','Two steps started, one awaited. The other has nothing following it.'],
  ['foreign-async','A step discovered only after non-Inngest async work.'],
  ['sleep-in-branch','A sleep inside one branch while the other keeps working.'],
  ['invoke','step.invoke(), whose child run is its own substance.'],
  ['wait','A wait that matched. The bar is blue while it is open, because nothing has decided it.'],
  ['wait-timeout','A wait nothing satisfies. The run carries on: a timeout is a result.'],
  ['retry','A step that threw, backed off and returned on the second attempt. The backoff is a bar of its own.'],
  ['blocked','Two runs on a limit of one, so the second spends real time queued rather than executing.'],
  ['cancelled','Cancelled while parked on a wait. A row that was open is neither succeeded nor failed.'],
];

// The source of each shape, read from the file that defines it rather than
// transcribed -- so the code on the page cannot drift from the run beside it.
const CODE=fixtureCode();

const F={};
for(const [id,cap] of PAIRS){
  F[id]={cap, code:CODE[id]||'', modes:[
    {mode:'checkpointing',    svg:capture('cp-'+id,   id+': with checkpointing')},
    {mode:'no checkpointing', svg:capture('nocp-'+id, id+': without checkpointing')},
  ]};
}

fs.writeFileSync(HERE+'batch1.json',JSON.stringify(F));
console.log('ok', Object.keys(F).join(' '));
