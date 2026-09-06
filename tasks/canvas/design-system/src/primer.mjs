const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,C,EV} from './micro.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";

/**
 * The primer: the four ideas the rest of the page is built from, each one
 * figure and one line. Drawn rather than described wherever a drawing will do.
 */
const O={};

const label=(p,t,dy=14)=>
  `<text x="${px(p).toFixed(1)}" y="${cy(0)+dy}" ${MONO} font-size="6.5" fill="${C.mut}" text-anchor="middle">${t}</text>`;

// 1. Moments. A distributed trace is not a picture, it is a list of things
//    that happened, each with a time.
O.events=fig(
  [{n:'', segs:[], dots:[{p:0,c:EV.queued},{p:34,c:EV.started},{p:86,c:EV.ok}]}],
  label(0,'queued')+label(34,'started')+label(86,'ok'),
  'three recorded moments','',{pad:14});

// 2. Intervals. The gap between two moments is the thing you actually want.
O.durations=fig(
  [{n:'', segs:[['idle',0,34],['good',34,52]]}],
  label(17,'queued')+label(60,'step.run()'),
  'the same moments with the time between them','',{pad:14});

// 3. The lifecycle. Every step in a durable system has this shape, however
//    many attempts it took.
const LC=[['idle',0,10,'queued'],['bad',10,20,'attempt 1'],['backoff',30,16,'backoff'],
          ['idle',46,6,'queued'],['good',52,34,'attempt 2']];
O.lifecycle=fig(
  [{n:'a step', segs:LC.map(([k,x,w])=>[k,x,w])}],
  LC.map(([,x,w,t])=>label(x+w/2,t)).join(''),
  'one step, two attempts','',{pad:14});

// 4. Hollow or filled — has this finished?
O.resolved=fig([
  {n:'running',   segs:[['idle',0,10],['running',10,76]],
   dots:[{p:0,c:EV.queued},{p:10,c:EV.started}], note:'not resolved'},
  {n:'succeeded', segs:[['idle',0,10],['good',10,76]], note:'resolved'},
  {n:'failed',    segs:[['idle',0,10],['bad',10,76]],  note:'resolved'},
],'','hollow until it resolves');

// 5. Solid or hatched — was your app running?
O.substance=fig([
  {n:'step.run()', segs:[['idle',0,8],['good',8,78]],   note:'solid'},
  {n:'step.sleep()',segs:[['idle',0,8],['waitok',8,78]],note:'hatched'},
  {n:'queued',    segs:[['idle',0,86]], dots:[{p:0,c:EV.queued}], note:'hatched'},
],'','solid is your compute');

fs.writeFileSync(HERE+'primer.json',JSON.stringify(O));
console.log('primer ok');
