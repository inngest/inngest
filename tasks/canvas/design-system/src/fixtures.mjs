const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,EV} from './micro.mjs';
import {autoDots,fillGaps} from './vocabulary.mjs';

/**
 * Captured fixtures, scrubbable.
 *
 * Each fixture is the finished trace at its measured proportions. A frame is
 * that trace clipped at a moment, with two adjustments that make the clipped
 * version honest rather than merely shorter:
 *
 *   - an interval still running has no outcome yet, so it takes the
 *     in-progress substance: a step.run() that will return and one that will
 *     throw are identical until they resolve;
 *   - a row that has not started yet is not drawn at all, because its span
 *     does not exist yet.
 *
 * The axis rescales so the elapsed part always fills the plot, which is what a
 * live trace does: the trace fills its area and the axis is what moves.
 */

const IN_PROGRESS={good:'running', bad:'running', waitout:'wait', waitok:'wait'};
const inProgress=k=>IN_PROGRESS[k]||k;

function at(rows, t){
  return rows.map(r=>{
    if(r.run) return {...r, to:Math.min(t, r.end),
      resolved:(r.resolvedAt!=null && t>=r.resolvedAt) ? r.resolvedAs : null};
    const full=fillGaps((r.segs||[]).map(([k,x,w])=>({kind:k,x,w})));
    const marks=autoDots(full);
    const segs=[];
    for(const g of full){
      if(g.x>=t) continue;
      const w=Math.min(g.w, t-g.x);
      segs.push([w<g.w?inProgress(g.kind):g.kind, g.x, w]);
    }
    if(!segs.length) return null;
    return {...r, segs, dots:marks.filter(d=>d.p<=t+0.001)};
  }).filter(Boolean);
}

/**
 * The moments worth stopping at are the trace's own: every point where a span
 * is created, starts executing, or resolves. Sampling time uniformly would
 * spend most of its frames on a sleep and none on the three things that
 * actually happened.
 */
function moments(rows){
  const ts=new Set();
  for(const r of rows){
    if(r.run){ for(const v of r.intervals){ ts.add(v.a); ts.add(v.b); } ts.add(r.end); continue; }
    for(const [,x,w] of r.segs){ ts.add(+x.toFixed(3)); ts.add(+(x+w).toFixed(3)); }
  }
  return [...ts].sort((p,q)=>p-q);
}

/** What a bar of this kind beginning actually means. */
const VERB={disc:'reported by a discovery request', child:'child run started',
            waitok:'waiting', wait:'waiting', backoff:'backing off'};
const say=(n,k)=>k==='child'?n+': child run started':k==='disc'?n+': discovery request started':n+' '+(VERB[k]||'started');

/** What changed at this moment, said the way the UI would say it. */
function label(rows, e){
  const eq=(p,q)=>Math.abs(p-q)<0.001;
  const started=[], ended=[], queued=[], retried=[];
  for(const r of rows){
    if(r.run) continue;
    const segs=r.segs;
    segs.forEach(([k,x,w],i)=>{
      if(eq(x,e)){
        if(k==='idle') queued.push(r.n);
        else if(k==='backoff') retried.push(r.n);
        else if(k!=='idle') started.push([r.n,k]);
      }
      if(eq(x+w,e) && i===segs.length-1) ended.push([r.n,k]);
    });
  }
  const bits=[];
  queued.forEach(n=>bits.push(n+' enqueued'));
  started.forEach(([n,k])=>bits.push(say(n,k)));
  retried.forEach(n=>bits.push(n+' backing off'));
  ended.forEach(([n,k])=>bits.push(n+(k==='bad'?' failed':k==='waitout'?' timed out':' resolved')));
  return bits.length?bits.slice(0,2).join(', '):'in progress';
}

const EPS=0.15;
const GAP=6;         // a stretch longer than this counts as dead time
const MIN=2, MAX=8;  // no stretch gets less scrubber than MIN or more than MAX

/**
 * An elastic scrub axis.
 *
 * A run is mostly waiting. Spending the scrubber in proportion to real time
 * gives 96% of its length to one sleep and piles every event into the last few
 * pixels. Each stretch between two events instead gets its real share clamped
 * to a floor and a ceiling: empty stretches fast-forward, short ones get room
 * to be grabbed. Reported times never change; only where the handle sits does.
 */
function elastic(M){
  const segs=[];
  for(let i=0;i<M.length-1;i++){
    const a=M[i], b=M[i+1], w=b-a;
    segs.push({a,b,w, alloc: Math.min(MAX, Math.max(MIN, w)), cut: w>GAP});
  }
  const total=segs.reduce((s,x)=>s+x.alloc,0)||1;
  let acc=0;
  for(const s of segs){ s.p0=+(acc/total*100).toFixed(3); acc+=s.alloc; s.p1=+(acc/total*100).toFixed(3); }
  return segs;
}
const posOf=(segs,t)=>{
  if(t<=segs[0].a) return 0;
  for(const s of segs) if(t<=s.b) return s.p1===s.p0?s.p0:s.p0+((t-s.a)/(s.b-s.a))*(s.p1-s.p0);
  return 100;
};

/**
 * Fixtures ship as data, not as pictures.
 *
 * The page computes the state at whatever moment the handle is on, so there is
 * an exact frame for every pixel of the scrubber rather than a sampled one
 * every few. It also means one trace per fixture on the wire instead of five
 * hundred drawings of it.
 */
const pack=(f)=>{
  const rows=f.rows.map(r=>{
    if(r.run) return {run:1, end:r.end, iv:r.intervals.map(v=>[v.a,v.b,v.ok===false?0:v.ok==='stop'?2:1]),
                      ra:r.resolvedAt==null?null:r.resolvedAt, rs:r.resolvedAs||null};
    const full=fillGaps(r.segs.map(([k,x,w])=>({kind:k,x,w})));
    return {n:r.n, segs:full.map(g=>[g.kind,+g.x.toFixed(3),+g.w.toFixed(3)]),
            marks:autoDots(full).map(d=>[d.c,+d.p.toFixed(3)])};
  });
  const M=moments(f.rows).map(e=>Math.max(e,EPS));
  const bounds=[...new Set([0,...M,100])].sort((x,y)=>x-y);
  const segs=elastic(bounds);
  const events=M.map(t=>({t:+t.toFixed(2), p:+posOf(segs,t).toFixed(3), text:label(f.rows, t===EPS?0:+t.toFixed(2))}));
  events.unshift({t:0,p:0,text:'run enqueued, first step not known yet'});
  return {id:f.id, title:f.title, note:f.note, rowCount:f.rows.length, rows, events,
          axis:segs.map(s=>[s.a,s.b,s.p0,s.p1]),
          breaks:segs.filter(s=>s.cut).map(s=>({p0:s.p0,p1:s.p1}))};
};

// ---------------------------------------------------------------------------
// Proportions come from the captured fixtures, unchanged.

const F=[];

// step: three steps around a 2s sleep. The sleep is 96% of the run.
F.push({id:'step', title:'step',
  note:'Three steps around a step.sleep(). Each step resolving causes a request to your app to report what runs next; that request is the short blue interval at the head of the next row. At real proportions it is 1ms against a 2s sleep, so it is only visible while the axis is still short.',
  rows:[
    {run:true, end:99.7, intervals:[{a:0,b:0.5,ok:true},{a:0.5,b:0.55,ok:true},{a:96.5,b:96.55,ok:true},
                                    {a:96.9,b:97.4,ok:true},{a:99.1,b:99.7,ok:true}],
     resolvedAt:99.7, resolvedAs:EV.ok},
    {n:'first step',   segs:[['good',0,0.5]]},
    {n:'for 2s',       segs:[['disc',0.5,0.05],['waitok',0.55,95.95]]},
    {n:'second step',  segs:[['disc',96.5,0.05],['idle',96.55,0.35],['good',96.9,0.5]]},
    {n:'Finalization', segs:[['idle',97.4,1.7],['good',99.1,0.6]]},
  ]});

// v4sequential: an SDK that can report batches, on a run where nothing is
// parallel. No ribbon anywhere is the correct render.
F.push({id:'v4sequential', title:'v4sequential',
  note:'A chain on an SDK that can report batches. Every request reported exactly one step, so no request gets a row of its own and there is no ribbon anywhere. Each one is the blue interval at the head of the step it reported.',
  rows:[
    {run:true, end:100, intervals:[{a:0.8,b:1.2,ok:true},{a:1.2,b:1.25,ok:true},{a:98.2,b:98.25,ok:true},
                                    {a:99.3,b:99.7,ok:true},{a:99.7,b:100,ok:true}],
     resolvedAt:100, resolvedAs:EV.ok},
    {n:'first step',   segs:[['idle',0,0.8],['good',0.8,0.4]]},
    {n:'for 2s',       segs:[['disc',1.2,0.05],['waitok',1.25,96.95]]},
    {n:'second step',  segs:[['disc',98.2,0.05],['idle',98.25,1.05],['good',99.3,0.4]]},
    {n:'Finalization', segs:[['good',99.7,0.3]]},
  ]});

// invoke: a child run drawn as its own substance inside the parent's row.
F.push({id:'invoke', title:'invoke',
  note:'step.invoke(). The row opens with queue time, then the blue request that reported the step, then the child run in its own colour on this run’s axis. The child is 61% of the parent and the parent does nothing while it runs.',
  rows:[
    {run:true, end:99, intervals:[{a:0,b:0.5,ok:true},{a:16.5,b:17.1,ok:true},{a:82.7,b:83.4,ok:true},{a:98.3,b:99,ok:true}],
     resolvedAt:99, resolvedAs:EV.ok},
    {n:'before',       segs:[['good',0,0.5]]},
    {n:'call child',   segs:[['idle',1,15.5],['disc',16.5,0.6],['child',17.1,61]]},
    {n:'after',        segs:[['idle',78.1,4.6],['good',82.7,0.7]]},
    {n:'Finalization', segs:[['idle',84.5,13.8],['good',98.3,0.7]]},
  ]});

fs.writeFileSync(HERE+'fixtures.json',JSON.stringify(F.map(pack)));
const J=JSON.parse(fs.readFileSync(HERE+'fixtures.json','utf8'));
console.log('fixtures',J.length,'| events',J.map(x=>x.events.length).join(','),
            '| breaks',J.map(x=>x.breaks.length).join(','),
            '| bytes',JSON.stringify(J).length);
