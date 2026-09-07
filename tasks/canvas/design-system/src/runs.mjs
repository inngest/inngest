/**
 * A captured run, turned into rows of moments.
 *
 * The fixtures used to be proportions typed out by hand from these same
 * payloads — a second copy of a measurement, which is the defect this whole
 * artifact keeps finding. They are read now.
 *
 * The payload is what the API returns: a root span for the run, one span per
 * step (plus one per attempt), and a list of discovery requests each carrying
 * the ids of the steps it planned. That last list is the relationship the
 * drawing needs and cannot infer — which request reported which steps — so
 * ribbons and cables come out of the capture rather than out of coincident
 * timings.
 */
import fs from 'fs';
import * as R from './rules.mjs';

const FIXTURES=new URL('../../../../ui/packages/components/src/RunDetailsV4/canvas/__fixtures__/',
  import.meta.url).pathname;

const ms=t=>t==null?null:Date.parse(t);

/** What a span's status means as a closing moment. */
const OUTCOME={
  COMPLETED:'ok', FAILED:'failed', CANCELLED:'cancelled',
  TIMED_OUT:'timeout', WAITING:null, RUNNING:null, QUEUED:null,
};

/** What kind of row a step is, from what the SDK called. */
const KIND={
  RUN:'step', SLEEP:'wait', WAIT_FOR_EVENT:'wait', WAIT_FOR_SIGNAL:'wait',
  INVOKE:'child', SEND_EVENT:'step', AI_GATEWAY:'step',
};

/** Spans the trace carries that are not rows of the run. */
const INTERNAL=/^executor\./;

export function loadRun(id){
  const j=JSON.parse(fs.readFileSync(FIXTURES+id+'.json','utf8'));
  const t=j.run.trace;
  const t0=ms(t.queuedAt), t1=ms(t.endedAt)||ms(t.startedAt)||t0;
  let total=Math.max(1, t1-t0);
  const at=v=>{ const n=ms(v); return n==null?null:n-t0; };

  // One row per step. A step has a span of its own and a span per attempt; the
  // step's span carries the queue, the last attempt carries the execution.
  const byStep=new Map();
  for(const s of (t.childrenSpans||[])){
    if(INTERNAL.test(s.name||'')) continue;
    const key=s.stepID || s.name;
    if(!byStep.has(key)) byStep.set(key,[]);
    byStep.get(key).push(s);
  }

  // Which discovery planned which step, and when it ran.
  const planner=new Map();
  for(const d of (t.discoveries||[]))
    for(const sid of (d.plannedStepIDs||[])) planner.set(sid, d);

  const rows=[];
  for(const [key,spans] of byStep){
    const first=spans.reduce((m,s)=>ms(s.queuedAt)<ms(m.queuedAt)?s:m);
    const last =spans.reduce((m,s)=>ms(s.endedAt||s.startedAt)>ms(m.endedAt||m.startedAt)?s:m);
    const exec =spans.reduce((m,s)=>ms(s.startedAt)>ms(m.startedAt)?s:m);
    const name=first.name;
    const isFin=name==='Finalization';
    const kind=KIND[first.stepOp]||'step';

    const q=at(first.queuedAt), st=at(exec.startedAt), en=at(last.endedAt);
    const out=OUTCOME[last.status];
    const d=planner.get(first.stepID);

    /**
     * Does the request that reported this step get a bar of its own?
     *
     * Only when it was a SEPARATE EXECUTION. Where the SDK reported and ran in
     * one request the two spans cover the same instant, and drawing a discovery
     * bar in front of the step would be drawing one execution as two. That is
     * the rule the Concepts page states; here it is a fact in the capture
     * rather than a judgement, because both spans carry their own timestamps.
     */
    const sep = d && !isFin && at(d.endedAt)!=null && st!=null && at(d.endedAt) <= st;
    const moments=[];
    if(sep){
      moments.push(['queued', at(d.queuedAt)]);
      moments.push(['started', at(d.startedAt), 'disc']);
      moments.push(['ok', at(d.endedAt)]);
    }
    // planned, where one request reported this step alongside others
    const multi=d && (d.plannedStepIDs||[]).length>1;
    const qq=sep ? Math.max(q, at(d.endedAt)) : q;
    if(qq!=null && (st==null || qq<=st)) moments.push([multi?'planned':'queued', qq]);
    if(st!=null) moments.push(['started', st]);
    if(out && en!=null && (moments.length===0 || en>=moments[moments.length-1][1]))
      moments.push([out, en]);

    // Sorted by when the STEP was queued, not by when the request that
    // reported it was: a row belongs where its own work sits.
    rows.push({_q:q==null?0:q, n:name, kind, at:moments, reported:sep||undefined,
               end:out?undefined:total, _stepID:first.stepID, _planner:d&&d.spanID});
  }

  rows.sort((a,b)=>a._q-b._q);

  // Steps a single request reported together are one ribbon; the capture says
  // which, so nothing has to be inferred from when they happened to be queued.
  for(const d of (t.discoveries||[])){
    const ids=d.plannedStepIDs||[];
    if(ids.length<2) continue;
    const mem=rows.filter(r=>ids.includes(r._stepID));
    if(mem.length>1){ mem[0].reports=mem.slice(1).map(r=>r.n); }
  }
  rows.forEach(r=>{ delete r._stepID; delete r._planner; delete r._q; });

  /**
   * The opening queue, named rather than drawn.
   *
   * Done here, on real timestamps, because that is the only place it can be
   * named: after the elastic pass the axis is no longer linear and the same
   * measurement reads as a different duration.
   */
  let lead='';
  if(R.FEAT.trim){
    let first=Infinity;
    for(const r of rows) for(const [k,x] of r.at) if(k==='started') { first=Math.min(first,x); break; }
    if(first>0 && first<Infinity){
      lead=R.human(first);
      const shift=mo=>mo.length===3?[mo[0],mo[1]-first,mo[2]]:[mo[0],mo[1]-first];
      for(const r of rows){
        const s=r.at.map(shift);
        const kept=s.filter(mo=>mo[1]>=0);
        // a row already queued when the drawing begins keeps the state it was in
        const open=(kept.length && kept[0][1]===0)?null:s.filter(mo=>mo[1]<0).pop();
        r.at=(open?[[...open].map((v,j)=>j===1?0:v)]:[]).concat(kept);
        if(r.end!=null) r.end=Math.max(0,r.end-first);
      }
      total-=first;
    }
  }
  return {id, ms:total, rows, lead};
}
