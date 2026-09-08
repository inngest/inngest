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
 * ## A discovery with no window
 *
 * Fixed at the source, and the fallbacks below are kept as guards rather than
 * as workarounds. A discovery used to report `startedAt === endedAt` whenever
 * the request it covered had planned a sleep: the sleep's own "starts when it
 * is queued" timestamp was being merged onto the EXECUTION span along with the
 * rest of the op's attributes, overwriting the moment the request began with
 * the moment it ended. Captures taken before that fix have no window on those
 * requests and cannot be given one.
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
/** The span that covers a request which produced no step of its own. */
const NONSTEP='executor.nonstep';

export function loadRun(id){
  const j=JSON.parse(fs.readFileSync(FIXTURES+id+'.json','utf8'));
  const t=j.run.trace;
  const t0=ms(t.queuedAt), t1=ms(t.endedAt)||ms(t.startedAt)||t0;
  let total=Math.max(1, t1-t0);
  const at=v=>{ const n=ms(v); return n==null?null:n-t0; };
  const runStart=at(t.startedAt);

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

  /**
   * One request, drawn once.
   *
   * A discovery that planned several steps is ONE execution, and every row it
   * planned was going to draw it -- the same blue bar, at the same instants, on
   * three rows, which says three requests ran where one did. The ribbon is
   * already the thing that says they share a request; the bar only has to
   * appear on one of them.
   *
   * Which one is decided by name, lexicographically, so it does not move when
   * the rows do: sorting by when they started would hand the bar to whichever
   * of a parallel batch happened to be picked up first, and it would land
   * somewhere different on the next capture of the same function.
   */
  const nameOfStep=new Map();
  for(const [key,spans] of byStep){
    const first=spans.reduce((m,sp)=>ms(sp.queuedAt)<ms(m.queuedAt)?sp:m);
    if(first.stepID) nameOfStep.set(first.stepID, first.name);
    nameOfStep.set(key, first.name);
  }
  const drawsRequest=new Map();   // discovery spanID -> the one row that draws it
  for(const d of (t.discoveries||[])){
    const names=(d.plannedStepIDs||[]).map(id=>nameOfStep.get(id)).filter(Boolean);
    if(names.length) drawsRequest.set(d.spanID, names.slice().sort()[0]);
  }

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
    // Was this step PLANNED by a separate request, and does this row draw it?
    // Every planned row gets the moment; only one of them gets the bar.
    const planned = d && !isFin && at(d.endedAt)!=null && st!=null && at(d.endedAt) <= st;
    const sep = planned && drawsRequest.get(d.spanID) === name;
    const moments=[];
    /**
     * Whether the instant this row OPENS on reached us by checkpoint.
     *
     * A row borrows that instant from whatever the request did before it, so a
     * row that is not itself checkpointed can still open on a checkpointed
     * moment -- `for 2s` begins where `first step` ended, and that end came
     * from a checkpoint. The ring follows where a timestamp came from, not
     * which row it landed on.
     */
    let priorCp=false, priorAt=null;
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
      // Only work the request actually RAN counts -- a span that merely ENDED
      // inside the window was running long before it, a sleep most often, and
      // taking its end as the request's earlier work put the request after a
      // wait it had nothing to do with.
      // ...using the request's own start only where it records a usable one:
      // a discovery whose startedAt equals its endedAt has no window, and
      // taking that as the floor excludes the very work it ran.
      const from=(ds!=null && ds>dq && ds<de) ? ds : dq;
      let prior=null;
      for(const o of (t.childrenSpans||[])){
        if(INTERNAL.test(o.name||'')) continue;
        if((o.stepID||o.name)===key) continue;
        const a=at(o.startedAt), b=at(o.endedAt);
        if(a==null||b==null||from==null||de==null) continue;
        if(a>=from && b<=de && (prior==null||b>prior)){ prior=b; priorCp=!!o.isCheckpoint; }
      }
      if(prior==null){
        moments.push(['queued', dq]);
        moments.push(['started', ds, 'disc']);
      }else{
        priorAt=Math.min(Math.max(prior,dq), de);
        moments.push(['started', priorAt, 'disc']);
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
    const ran = planned ? null : (t.discoveries||[]).find(x=>{
      const a=at(x.queuedAt), b=at(x.endedAt);
      return a!=null && b!=null && st!=null && a<=st && st<=b; });
    let ranFrom=null;
    if(ran){
      ranFrom=at(ran.queuedAt);
      const rstart=at(ran.startedAt);
      const from=rstart!=null&&rstart>ranFrom ? rstart : ranFrom;
      for(const o of (t.childrenSpans||[])){
        if(INTERNAL.test(o.name||'')) continue;
        if((o.stepID||o.name)===key) continue;
        const a=at(o.startedAt), b=at(o.endedAt);
        // Same rule as `prior`: the request ran it only if it began after the
        // request did.
        if(a!=null && b!=null && a>=from && b>ranFrom && b<=st){ ranFrom=b; priorCp=!!o.isCheckpoint; }
      }
    }
    /**
     * The lead-in to a checkpointed step is the REQUEST, and most of it is
     * compute.
     *
     * The SDK is running your function the whole time it is between two inline
     * steps -- that is your app executing, not the step waiting its turn -- so
     * drawing it hatched said the opposite of what happened. `emit` runs three
     * steps in one request and had five and six millisecond queues between
     * them; there was no queue, the SDK was working.
     *
     * So the lead-in splits at the moment the request started executing:
     * waiting before it, the SDK's own work after it. A step that is not the
     * first in its request has no waiting part at all -- the request was
     * already running when the previous step returned.
     */
    let qq=planned ? at(d.endedAt) : (ranFrom!=null ? Math.min(ranFrom, st) : q);
    if(ran && ranFrom!=null && st!=null && ranFrom<st){
      const rstart=at(ran.startedAt), rend=at(ran.endedAt);
      // A request whose startedAt equals its endedAt recorded no window at
      // all, so it cannot say when it began; the run's own start is the next
      // best thing, and for the first request it is the same instant.
      let rs = (rstart!=null && !(rend!=null && rstart===rend)) ? rstart : runStart;
      // Already executing when this row's lead-in began: there is nothing to
      // wait for, the whole lead-in is the SDK working towards this step.
      if(rs!=null && rs<=ranFrom) rs=null;
      priorAt=ranFrom;
      if(rs!=null){
        moments.push(['queued', ranFrom]);
        if(rs<st){                       // ...and then the SDK's own work
          moments.push(['started', rs, 'disc']);
          moments.push(['ok', st]);
        }
      }else{
        moments.push(['started', ranFrom, 'disc']);
        moments.push(['ok', st]);
      }
      qq=null;
    }
    if(qq!=null && (st==null || qq<=st)) moments.push([planned?'planned':'queued', qq]);
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
    const cpAt = new Set();
    if(isCp && st!=null) for(const mo of moments) if(mo[1]>=st) cpAt.add(mo[1]);
    if(priorCp && priorAt!=null) cpAt.add(priorAt);
    const cp = cpAt.size ? [...cpAt] : undefined;

    rows.push({_q:q==null?0:q, n:name, kind, at:moments, cp, reported:sep||undefined,
               end:out?undefined:total, _stepID:first.stepID, _planner:d&&d.spanID});
  }

  rows.sort((a,b)=>a._q-b._q);

  /**
   * The finalization, where the run did not make a request for it.
   *
   * v3 finalizes in a request of its own and the executor gives that request a
   * span named `Finalization`. v4 does not: the function returns inside the
   * last step's request, so `RunComplete` comes back in that response and
   * there is no separate span to name. The work is still there -- five to ten
   * milliseconds of it, the SDK running your function to its return -- inside
   * the `executor.nonstep` span that covers the last request and carries the
   * run's output.
   *
   * So the row is read off that span rather than invented: it starts where the
   * last step ended and ends where the run did, both measured. Where the run
   * DID make a finalization request there is a span called Finalization
   * already and this does nothing.
   */
  if(!rows.some(r=>r.n==='Finalization')){
    const last=rows.reduce((m,r)=>{
      for(const [,x] of (r.at||[])) m=Math.max(m,x);
      return m; }, 0);
    // The request that was open when the run ended, and nothing else: a
    // nonstep span that stops short of the run is some earlier request.
    const tail=(t.childrenSpans||[])
      .filter(sp=>(sp.name||'')===NONSTEP && at(sp.endedAt)!=null)
      .find(sp=>Math.abs(at(sp.endedAt)-total)<2);
    if(tail && total-last > 0.5){
      // It opens on the instant the last step ended, so where that step was
      // checkpointed the finalization opens on a checkpointed instant -- the
      // same borrowing every other row does. The resolution is not: the run
      // completing came back in the response.
      const lastCp=(t.childrenSpans||[]).some(sp=>sp.isCheckpoint &&
        at(sp.endedAt)!=null && Math.abs(at(sp.endedAt)-last)<1e-9);
      rows.push({_q:last, n:'Finalization', kind:'step',
        at:[['started',last],['ok',total]], cp:lastCp?[last]:undefined});
    }
  }

  // Steps a single request reported together are one ribbon; the capture says
  // which, so nothing has to be inferred from when they happened to be queued.
  for(const d of (t.discoveries||[])){
    const ids=d.plannedStepIDs||[];
    if(ids.length<2) continue;
    const mem=rows.filter(r=>ids.includes(r._stepID));
    // The link goes on the row that DRAWS the request, not on whichever came
    // first in row order. The ribbon is hung at that row's resolution -- the
    // instant the request reported them -- and only the drawing row has that
    // moment; on any of the others it is their own step resolving, which is
    // somewhere else entirely.
    if(mem.length>1){
      const lead=mem.find(r=>r.n===drawsRequest.get(d.spanID)) || mem[0];
      lead.reports=mem.filter(r=>r!==lead).map(r=>r.n);
    }
  }
  rows.forEach(r=>{ delete r._stepID; delete r._planner; delete r._q; });

  // The run as it happened. What to hide is a drawing decision, made where
  // the drawing is: trimming here would bake it in and the toggle could not
  // undo it.
  return {ms:total, rows};
}
