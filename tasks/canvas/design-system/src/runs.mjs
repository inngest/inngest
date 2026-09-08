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
 * ## Why a v4 run has fewer requests than it has steps
 *
 * Confirmed by instrumenting the executor, because none of it is visible in
 * the payload. `v4sequential` -- three steps and a finalize -- makes exactly
 * TWO requests:
 *
 *     request 1   checkpoint: StepRun("first step")     <- out of band
 *                 response:   1 op, Sleep("2s")
 *     ... 2s ...
 *     request 2   checkpoint: StepRun("second step")
 *                 response:   1 op, RunComplete
 *
 * A response still carries ONE op, as it always did. What is new in v4 is
 * that a `step.run` the SDK can execute inline is CHECKPOINTED -- reported
 * out of band while the request is still open -- so the SDK carries on into
 * the next step and the response is the op it could not run itself. One
 * request therefore both runs a step and reports the next one.
 *
 * Two things follow, and both are why this file looks the way it does:
 *
 *   - A checkpointed step has no request of its own and no queue of its own:
 *     its span's `queuedAt` is the instant the SDK ran it. The wait it really
 *     had belongs to the request that carried it, which is what `ranFrom`
 *     below recovers.
 *   - The request that plans the sleep is the SAME request that ran
 *     `first step`, so its `queuedAt` is legitimately the run's. Only the
 *     part of it after the step belongs to the sleep, which is what `prior`
 *     below is for.
 *
 * ## What the payload still gets wrong
 *
 * One thing, and it wants fixing at the source: **a discovery reports
 * `startedAt === endedAt`**. The executor knows when the request began -- it
 * sets it on the execution span at creation -- but the stored span comes back
 * with the start replaced by the end, so the request has no window to draw.
 * True of `v4sequential` and of `step`.
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
       * A request can RUN steps before it plans the next one.
       *
       * In v4 a sequential `step.run` is checkpointed out of band, so the SDK
       * carries straight on and the same request goes on to report the sleep
       * (see the note at the top of this file). That request's queue is the
       * run's queue and its first stretch of work is `first step`, neither of
       * which has anything to do with the sleep -- so drawing the request from
       * its `queuedAt` put all of it in the sleep's row, which then appeared
       * to have been waiting since the run was enqueued, through another
       * step's whole execution.
       *
       * The part of the request that produced THIS step is what came after the
       * last thing it ran. There is no queue in front of it because the step
       * did not exist to be queued.
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
    /**
     * Which of this row's moments arrived by CHECKPOINT.
     *
     * The span carries `isCheckpoint`, and it covers the STEP's moments only:
     * where a separate request planned the step, that request's own moments
     * came back in the response like any other and are not marked. So the
     * checkpointed instants are the ones after the request's, which is exactly
     * the tail of the list.
     */
    const isCp = spans.some(sp=>sp.isCheckpoint);
    const cp = (isCp && st!=null)
      ? moments.filter(mo=>mo[1]>=st).map(mo=>mo[1])
      : undefined;

    rows.push({_q:q==null?0:q, n:name, kind, at:moments, cp, reported:sep||undefined,
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
