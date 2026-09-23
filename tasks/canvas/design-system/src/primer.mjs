const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,C,EV,setLiteral} from './micro.mjs';
setLiteral(true);
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
  // Timestamps before they are anything else: this figure is the moments, so
  // it draws them and nothing between them.
  [{n:'', at:[['queued',0],['started',34],['ok',86]], bars:false}],
  label(0,'queued')+label(34,'started')+label(86,'ok'),
  'three recorded moments','',{pad:14});

// 2. Intervals. The gap between two moments is the thing you actually want.
O.durations=fig(
  [{n:'', at:[['queued',0],['started',34],['ok',86]]}],
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
  {n:'running',   at:[['queued',0],['started',10]],end:86,
   note:'not resolved'},
  {n:'succeeded', at:[['queued',0],['started',10],['ok',86]], note:'resolved'},
  {n:'failed',    at:[['queued',0],['started',10],['failed',86]],  note:'resolved'},
],'','hollow until it resolves');

// 5. Solid or hatched — was your app running?
O.substance=fig([
  {n:'step.run()', at:[['queued',0],['started',8],['ok',86]],   note:'solid'},
  {n:'step.sleep()',at:[['queued',0],['started',8],['ok',86]],kind:'wait',note:'hatched'},
  {n:'queued',    at:[['queued',0]],end:86, note:'hatched'},
],'','solid is your compute');

fs.writeFileSync(HERE+'primer.json',JSON.stringify(O));
console.log('primer ok');
