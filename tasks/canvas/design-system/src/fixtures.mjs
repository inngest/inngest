const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {loadRun} from './runs.mjs';
import * as R from './rules.mjs';

/**
 * The scrubbable fixtures: a captured run, and the moments worth stopping at.
 *
 * Nothing here is drawn. This used to hand the page rows of BARS, an axis, a
 * list of compressed stretches and a row count, all computed here — and the
 * page had its own renderer to turn them back into a trace, which is why a
 * scrubbed frame's compressed stretch had no blur and its Run row never tore.
 * The page carries the real renderer now, so this carries only what a run is:
 * rows of moments, in milliseconds.
 */

/**
 * The moments worth stopping at are the trace's own: every point where a span
 * is created, starts executing, or resolves. Sampling time uniformly would
 * spend most of its frames on a sleep and none on the three things that
 * actually happened.
 */
function stopsOf(run){
  const ts=new Set([0, run.ms]);
  for(const r of run.rows){
    for(const [,x] of r.at) ts.add(+(+x).toFixed(3));
    if(r.end!=null) ts.add(+(+r.end).toFixed(3));
  }
  return [...ts].filter(v=>v>=0 && v<=run.ms).sort((a,b)=>a-b);
}

/** What changed at this moment, said the way the UI would say it. */
const SAID={
  queued:'enqueued', planned:'planned by a discovery request', held:'held by flow control',
  started:'started', retry:'threw, will retry', ok:'resolved', failed:'failed',
  timeout:'timed out', cancelled:'cancelled', done:'resolved',
};
function labelAt(run, t){
  const bits=[];
  for(const r of run.rows)
    for(const m of r.at)
      if(Math.abs(m[1]-t)<0.001 && SAID[m[0]]) bits.push(r.n+' '+SAID[m[0]]);
  return bits.length?bits.slice(0,2).join(', '):'in progress';
}

const FIXTURES=[
  {id:'step', title:'step',
   note:'Three steps around a step.sleep(). Each step resolving causes a request to your app to report what runs next; that request is the short blue interval at the head of the next row.'},
  {id:'v4sequential', title:'v4sequential',
   note:'A chain on an SDK that can report batches. Every request reported exactly one step, so no request gets a row of its own and there is no ribbon anywhere.'},
  {id:'invoke', title:'invoke',
   note:'step.invoke(). The row opens with queue time, then the blue request that reported the step, then the child run in its own colour on this run’s axis.'},
];

const pack=f=>{
  const run=loadRun(f.id);
  const stops=stopsOf(run);
  return {...f, ms:run.ms, lead:run.lead, rows:run.rows, stops,
          events:stops.map(t=>({t, text:labelAt(run,t)}))};
};

fs.writeFileSync(HERE+'fixtures.json',JSON.stringify(FIXTURES.map(pack)));
console.log('ok', FIXTURES.map(f=>f.id).join(' '));
