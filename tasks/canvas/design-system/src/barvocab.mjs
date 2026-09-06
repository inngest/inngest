const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {setNotes,fig,px,cy,dot,tag,EV,C,W,LBL,PLOT,ROW,TOP,HATCH,NOCOMPUTE,COMPUTE,BAR_INFO,FILL} from './micro.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";
const O={};

// Colour = what kind of work. Height = whether it is time on your app.
// Derived from the vocabulary, so a kind added there appears here.
const BARS=Object.entries(BAR_INFO).map(([k,[n,d]])=>
  [k,n,d,COMPUTE.has(k)?'compute':'not compute']);
O.key=(()=>{
  const groups=[['SDK executing: solid',BARS.filter(r=>r[3]==='compute')],
                ['not executing: hatched',BARS.filter(r=>r[3]!=='compute')]];
  let y=12,s='';
  for(const [title,rows] of groups){
    s+=`<text x="14" y="${y}" ${MONO} font-size="7" fill="${C.mut}" opacity=".65">${title}</text>`;
    y+=11;
    for(const [k,n,d] of rows){
      const h=7, nc=NOCOMPUTE.has(k);
      s+=(nc?`<rect x="16" y="${y-h/2}" width="34" height="${h}" rx="1.2" fill="url(#hx-${k})"/>`
             :`<rect x="16" y="${y-h/2}" width="34" height="${h}" rx="1.2" fill="${FILL[k]}"/>`)
       +`<text x="58" y="${y+2.5}" ${MONO} font-size="7.5" fill="${C.mut}">${n}</text>`
       +`<text x="148" y="${y+2.5}" ${MONO} font-size="7" fill="${C.mut}" opacity=".85">${d}</text>`;
      y+=15;
    }
    y+=6;
  }
  return `<svg viewBox="0 0 ${W} ${y}" role="img" aria-label="What the bar colours mean">`+HATCH+s+`</svg>`;
})();

// The height rule, on one run.
O.height=fig([
  {n:'a · owner',segs:[['disc',0,10],['idle',10,8],['good',18,30]],note:'discovery, then the step'},
  {n:'b · member',segs:[['idle',10,8],['good',18,26]],noHalo:[10],note:'planned'},
  {n:'c · seq',segs:[['idle',50,6],['good',56,26]],note:'discovery queued'},
  {n:'nap 2s',segs:[['idle',82,3],['waitok',85,12]],note:'no compute'},
],'','solid is your compute, hatched is not');

// A step resolving.
O.resolve=fig([
  {n:'running',segs:[['idle',0,10],['running',10,44]],dots:[{p:0,c:EV.queued},{p:10,c:EV.started}],note:'undecided'},
  {n:'succeeded',segs:[['idle',0,10],['good',10,50]],note:'50ms'},
  {n:'failed',segs:[['idle',0,10],['bad',10,38]],note:'failed'},
],'','a running bar resolves to green or red');

// Waits resolving.
O.waits=fig([
  {n:'sleep 2s',segs:[['idle',0,6],['waitok',6,54]],note:'cannot fail'},
  {n:'matched',segs:[['idle',0,6],['waitok',6,44]],note:'event arrived'},
  {n:'timed out',segs:[['idle',0,6],['waitout',6,64]],note:'nothing matched'},
  {n:'still open',segs:[['idle',0,6],['wait',6,88]],note:'undecided'},
],'','how waiting is going to go');

setNotes(true);
O.notes=fig([
  {n:'a',segs:[['idle',0,8],['good',8,34]],note:'34ms'},
  {n:'b',segs:[['idle',0,8],['bad',8,18],['backoff',26,10],['idle',36,4],['good',40,20]],note:'2 attempts · 20ms'},
  {n:'c',segs:[['idle',0,8],['good',8,52]],note:'planned 3rd'},
  {n:'nap 2s',segs:[['idle',0,4],['waitok',4,74]],note:'no compute'},
],'','annotations sit after the bar they describe');

fs.writeFileSync(HERE+'barvocab.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
