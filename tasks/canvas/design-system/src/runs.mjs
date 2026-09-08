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
 *
 * ## What the captures do not say
 *
 * Three gaps, all in the v4 payloads, all of which this file compensates for
 * rather than corrects -- they want fixing at the source:
 *
 *   - **Requests go unrecorded.** `v4sequential` lists two, for a run that
 *     made at least four (run `first step`; plan the sleep; run `second
 *     step`; finalize). `step` and `v4parallel` list every one of theirs.
 *     So a step can have no request in the payload at all, and the request
 *     that produced it has to be found by which window contains it.
 *   - **A request can carry the RUN's `queuedAt` instead of its own.**
 *     `v4sequential`'s surviving request reports 1ms -- the run's queue --
 *     though it cannot have been enqueued before the step that preceded it
 *     returned at 68ms.
 *   - **A request that plans a sleep records `startedAt === endedAt`.** True
 *     of `v4sequential` and of `step`, and of nothing else. There is no
 *     execution window, so that request cannot be drawn as one.
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
  // The FIRST request to report a step is the one that planned it. A request
  // that goes on to run a step lists it again, and last-write-wins pointed the
  // row at the request that ran it -- whose window ends after the step starts,
  // so the row lost its discovery bar and its planned mark entirely.
  const planner=new Map();
  for(const d of (t.discoveries||[]))
    for(const sid of (d.plannedStepIDs||[])) if(!planner.has(sid)) planner.set(sid, d);

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
      const dq=at(d.queuedAt), ds=at(d.startedAt), de=at(d.endedAt);
      /**
       * Where a request's own queue is not recorded, take the end of the work
       * that preceded it.
       *
       * A request is enqueued when the previous one returns, so the request
       * that plans a step cannot have been waiting through the execution of a
       * step that ran before it. The v4 capture reports the RUN's `queuedAt`
       * on the request that plans the sleep, which put all of that in the
       * sleep's row: it appeared to have been waiting since the run was
       * enqueued, through `first step`'s whole execution.
       *
       * So the request is drawn from the end of the last step that finished
       * inside its reported window. That is a floor, not the real queue -- the
       * queue between the two is real and is simply not in the data (see the
       * capture notes below). Where the request records a queue of its own,
       * which every other fixture does, this does not fire.
       */
      let prior=null;
      for(const o of (t.childrenSpans||[])){
        if(INTERNAL.test(o.name||'')) continue;
        if((o.stepID||o.name)===key) continue;
        const a=at(o.startedAt), b=at(o.endedAt);
        if(a==null||b==null||dq==null||de==null) continue;
        if(a>=dq && b<=de) prior=prior==null?b:Math.max(prior,b);
      }
      if(prior==null){
        moments.push(['queued', dq]);
        moments.push(['started', ds, 'disc']);
      }else{
        moments.push(['started', Math.min(Math.max(prior,dq), de), 'disc']);
      }
      moments.push(['ok', de]);
    }
    /**
     * A step reported by a separate request was PLANNED, and it was planned at
     * the instant that request reported it.
     *
     * `planned` used to be reserved for a request that reported several steps,
     * on the reasoning that one step needs no ribbon. But the mark is not about
     * the ribbon: it says this step was handed to the executor by a request
     * rather than run by one, which is the difference between a sleep and a
     * `step.run` in a sequential thread. Placing it where the request resolved
     * also collapses it with that resolution into the one circle it is -- they
     * were a millisecond apart and drew two overlapping marks.
     */
    /**
     * A step nobody planned was still QUEUED -- inside the request that ran it.
     *
     * A sequential `step.run` is discovered and run in one execution, so it
     * has no plan of its own, and its span's `queuedAt` is the instant the SDK
     * started running it rather than when it began waiting. The only record
     * that it waited at all is the request that did both: the run sat in the
     * queue, the executor picked it up, the SDK worked its way down to the
     * step. Without this the row opened on a queue of zero width and the first
     * step of every v4 capture appeared to start the moment the run did.
     *
     * Where the request ran SEVERAL steps the wait is only the part after the
     * previous one returned; the rest was that step's, not this one's.
     */
    const ran = sep ? null : (t.discoveries||[]).find(x=>{
      const a=at(x.queuedAt), b=at(x.endedAt);
      return a!=null && b!=null && st!=null && a<=st && st<=b; });
    let ranFrom=null;
    if(ran){
      ranFrom=at(ran.queuedAt);
      for(const o of (t.childrenSpans||[])){
        if(INTERNAL.test(o.name||'')) continue;
        if((o.stepID||o.name)===key) continue;
        const b=at(o.endedAt);
        if(b!=null && b>ranFrom && b<=st) ranFrom=b;
      }
    }
    const qq=sep ? at(d.endedAt) : (ranFrom!=null ? Math.min(ranFrom, st) : q);
    if(qq!=null && (st==null || qq<=st)) moments.push([sep?'planned':'queued', qq]);
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

  // The run as it happened. What to hide is a drawing decision, made where
  // the drawing is: trimming here would bake it in and the toggle could not
  // undo it.
  return {ms:total, rows};
}
