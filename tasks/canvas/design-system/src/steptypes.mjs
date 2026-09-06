const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,EV,px,cy,lineage} from './micro.mjs';
const E=86;

/**
 * One fragment per step type, with the states it reaches in the order it
 * reaches them: queued, in progress, then however it resolved, cancelled last.
 */
const GROUPS=[
 ['step.run', 'Attempts, backoff and the result are intervals in one row, not separate rows.', [
   ['queued',      [['idle',0,E]], 'queued'],
   ['running',     [['idle',0,10],['running',10,E-10]], 'open'],
   ['succeeded',   [['idle',0,10],['good',10,E-10]]],
   ['retried',     [['idle',0,8],['bad',8,20],['backoff',28,12],['idle',40,4],['good',44,E-44]]],
   ['failed',      [['idle',0,10],['bad',10,E-30]]],
   ['cancelled',   [['idle',0,10],['stopped',10,E-34]]],
 ]],

 ['step.sleep', 'Blue while it is running, like anything unresolved. It goes green because step.sleep() cannot fail: it elapses, or the run is cancelled.', [
   ['sleeping',    [['idle',0,6],['wait',6,E-6]], 'open'],
   ['finished',    [['idle',0,6],['waitok',6,E-6]]],
   ['cancelled',   [['idle',0,6],['waitstop',6,E-40]]],
 ]],

 ['step.waitForEvent', 'A timeout is a result, not a failure.', [
   ['waiting',     [['idle',0,6],['wait',6,E-6]], 'open'],
   ['matched',     [['idle',0,6],['waitok',6,E-26]]],
   ['timed out',   [['idle',0,6],['waitout',6,E-6]]],
   ['cancelled',   [['idle',0,6],['waitstop',6,E-30]]],
 ]],

 ['step.waitForSignal', 'Same states as step.waitForEvent(). Only what resolves it differs.', [
   ['waiting',     [['idle',0,6],['wait',6,E-6]], 'open'],
   ['received',    [['idle',0,6],['waitok',6,E-30]]],
   ['timed out',   [['idle',0,6],['waitout',6,E-6]]],
   ['cancelled',   [['idle',0,6],['waitstop',6,E-40]]],
 ]],

 ['step.invoke', 'The bar is the child run’s duration, drawn on this run’s axis.', [
   ['running',     [['idle',0,10],['child',10,E-10]], 'open'],
   ['finished',    [['idle',0,10],['child',10,E-24]]],
   ['failed',      [['idle',0,10],['child!',10,E-30]]],
   ['cancelled',   [['idle',0,10],['stopped',10,E-40]]],
 ]],

 ['step.sendEvent', 'The row reports how many runs the events started, and that count is the way into them.', [
   ['sent',        [['idle',0,10],['good',10,E-46]]],
   ['sent nothing',[['idle',0,10],['good',10,E-52]]],
   ['failed',      [['idle',0,10],['bad',10,E-52]]],
 ]],
];

const O={};
for(const [type,cap,states] of GROUPS){
  const rows=[];
  for(const [label,segs,mode,rowNote] of states){
    const r={n:label, segs, note:rowNote||''};
    // An open row stops at the start of its last bar: nothing has resolved it
    // yet, so there is no closing mark to draw.
    if(mode==='open'){ const l=segs[segs.length-1];
      r.dots=[{p:segs[0][1],c:EV.queued},{p:l[1],c:EV.started}]; }
    if(mode==='queued') r.dots=[{p:segs[0][1],c:EV.queued}];
    rows.push(r);
  }
  // step.sendEvent() is the one row that points out of the run: a spur to a
  // hollow ring and a count of the runs its events started.
  const opts = type==='step.sendEvent'
    ? {scale:1.6, over:(X,CY)=>lineage(X(E-46),CY(0),2)}
    : {};
  O[type]={svg:fig(rows,'',type+' states','',opts), cap};
}

fs.writeFileSync(HERE+'steptypes.json',JSON.stringify(O));
console.log('types',Object.keys(O).length);
