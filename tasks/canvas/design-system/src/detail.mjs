const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,tag,C,ROW} from './micro.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";

/**
 * The selected row, given its own axis.
 *
 * In the trace a row is scaled against the whole run, so a step that took 40ms
 * inside a two-minute run is a few pixels and its marks pile on top of each
 * other. Selecting it re-plots that one row across the full width using the
 * same vocabulary — same bars, same marks, same colours — and writes what each
 * piece is underneath. Nothing new to learn, and the main view stays as it is.
 */

// The row as it appears in the run: 6% of the width, marks overlapping.
const inTrace = fig([
  {n:'a',      segs:[['good',0,26]]},
  {n:'b',      segs:[['idle',26,4],['good',30,52]]},
  {n:'c',      segs:[['idle',82,1],['bad',83,1],['backoff',84,1],['good',85,1]],sel:true,note:'10.3s'},
], '', 'a short row inside a long run');

// The same row, re-plotted on its own axis.
const SEGS=[['idle',0,10,'queued','1.2s'],
            ['bad',10,20,'attempt 1','2.4s'],
            ['backoff',30,16,'backoff','1.9s'],
            ['idle',46,6,'queued','720ms'],
            ['good',52,34,'attempt 2','4.1s']];
const MARKS=[[0,'queued'],[10,'started'],[30,'retry'],[46,'queued'],[52,'started'],[86,'ok']];

const under=(()=>{
  let s='';
  // what each interval was, centred under it
  for(const [,x,w,name,dur] of SEGS){
    const mid=px(x+w/2);
    s+=`<text x="${mid.toFixed(1)}" y="${cy(0)+13}" ${MONO} font-size="6.5" fill="${C.mut}" text-anchor="middle">${name}</text>`;
    s+=`<text x="${mid.toFixed(1)}" y="${cy(0)+21}" ${MONO} font-size="6.5" fill="${C.ink2}" text-anchor="middle">${dur}</text>`;
  }
  // and what each mark is, on a second line with a tick up to it
  MARKS.forEach(([p,name],i)=>{
    const x=px(p), lo=i%2===1;
    s+=`<line x1="${x.toFixed(1)}" y1="${cy(0)+26}" x2="${x.toFixed(1)}" y2="${cy(0)+(lo?39:30)}" stroke="${C.idle}" stroke-width="1"/>`;
    s+=`<text x="${x.toFixed(1)}" y="${cy(0)+(lo?46:37)}" ${MONO} font-size="6.5" fill="${C.mut}" text-anchor="middle">${name}</text>`;
  });
  return s;
})();

const expanded = fig(
  [{n:'c',segs:SEGS.map(([k,x,w])=>[k,x,w])}],
  under, 'the selected row on its own axis', '', {pad:44});

fs.writeFileSync(HERE+'detail.json',JSON.stringify({
  frames:[{l:'in the trace',svg:inTrace},{l:'selected',svg:expanded}],
}));
console.log('detail ok');
