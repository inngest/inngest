const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,setFrame} from './micro.mjs';
import {loadRun} from './runs.mjs';
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
  ['parallel','A parallel batch has to be planned, so the fan-out is the same either way and only the step after it moves.'],
  ['chains','Parallel branches that are chains: planning and checkpointing in the same run.'],
  ['invoke','step.invoke(), whose child run is its own substance.'],
];

const F={};
for(const [id,cap] of PAIRS){
  F[id]=[
    {mode:'checkpointing',    cap, svg:capture('cp-'+id,   id+': with checkpointing')},
    {mode:'no checkpointing', cap, svg:capture('nocp-'+id, id+': without checkpointing')},
  ];
}

fs.writeFileSync(HERE+'batch1.json',JSON.stringify(F));
console.log('ok', Object.keys(F).join(' '));
