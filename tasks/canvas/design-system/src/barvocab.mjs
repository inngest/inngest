const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {setNotes,setLiteral,fig,px,cy,dot,tag,EV,C,W,LBL,PLOT,ROW,TOP,HATCH,NOCOMPUTE,COMPUTE,BAR_INFO,FILL} from './micro.mjs';
setLiteral(true);
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
  {n:'a · owner',at:[['queued',0],['started',3.5,'disc'],['ok',10],['planned',10],['started',18],['ok',48]],reported:1,note:'discovery, then the step'},
  {n:'b · member',at:[['planned',10],['started',18],['ok',44]],note:'planned'},
  {n:'c · seq',at:[['queued',50],['started',56],['ok',82]],note:'discovery queued'},
  {n:'nap 2s',at:[['queued',82],['started',85],['ok',97]],kind:'wait',note:'no compute'},
],'','solid is your compute, hatched is not');

// A step resolving.
O.resolve=fig([
  {n:'running',at:[['queued',0],['started',10]],end:54,note:'undecided'},
  {n:'succeeded',at:[['queued',0],['started',10],['ok',60]],note:'50ms'},
  {n:'failed',at:[['queued',0],['started',10],['failed',48]],note:'failed'},
],'','a running bar resolves to green or red');

// Waits resolving.
O.waits=fig([
  {n:'sleep 2s',at:[['queued',0],['started',6],['ok',60]],kind:'wait',note:'cannot fail'},
  {n:'matched',at:[['queued',0],['started',6],['ok',50]],kind:'wait',note:'event arrived'},
  {n:'timed out',at:[['queued',0],['started',6],['timeout',70]],kind:'wait',note:'nothing matched'},
  {n:'still open',at:[['queued',0],['started',6]],kind:'wait',end:94,note:'undecided'},
],'','how waiting is going to go');

setNotes(true);
O.notes=fig([
  {n:'a',at:[['queued',0],['started',8],['ok',42]],note:'34ms'},
  {n:'b',at:[['queued',0],['started',8],['retry',26],['queued',36],['started',40],['ok',60]],note:'2 attempts · 20ms'},
  {n:'c',at:[['queued',0],['started',8],['ok',60]],note:'planned 3rd'},
  {n:'nap 2s',at:[['queued',0],['started',4],['ok',78]],kind:'wait',note:'no compute'},
],'','annotations sit after the bar they describe');

fs.writeFileSync(HERE+'barvocab.json',JSON.stringify(O));
console.log('ok',Object.keys(O).join(' '));
