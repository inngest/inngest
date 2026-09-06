const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {panel,px,cy,C,W,LBL,PLOT,ROW,TOP,dot,EV} from './render.mjs';
const MONO="font-family='JetBrains Mono, ui-monospace, monospace'";
const COMPUTE='var(--compute)';

// Real measured compute intervals (start%, end%, outcome) — platform rows excluded.
const WIDE=[[24.9,26.1],[26.18,32.13],[26.18,31.97],[26.50,32.29],[26.34,31.97],[25.53,30.67],[25.69,30.83],
            [25.85,30.82],[25.20,29.20],[24.71,28.55],[24.88,28.55],[22.76,26.11],[24.55,26.92]]
            .map(([a,b])=>({a,b,ok:true}));
const MIXED=[{a:0.3,b:2.8,ok:true},{a:55.9,b:97.9,ok:true},{a:52.78,b:53.06,ok:true},{a:55.55,b:55.83,ok:false},{a:98.45,b:98.73,ok:true}];
const RETRY=[{a:0.3,b:1.2,ok:true},{a:49.6,b:50.2,ok:true},{a:0,b:0.28,ok:false},{a:49.15,b:49.43,ok:true},{a:50.75,b:51.03,ok:false},{a:99.61,b:99.89,ok:true}];

/** Slice the axis wherever any interval starts or ends. */
function slices(iv){
  const edges=[...new Set(iv.flatMap(i=>[i.a,i.b]))].sort((x,y)=>x-y);
  const out=[];
  for(let k=0;k<edges.length-1;k++){
    const a=edges[k], b=edges[k+1];
    const live=iv.filter(i=>i.a<=a && i.b>=b);
    if(live.length) out.push({a,b,n:live.length,ok:live.every(i=>i.ok),bad:live.every(i=>!i.ok)});
  }
  return out;
}
const maxN=s=>s.reduce((m,x)=>Math.max(m,x.n),1);

function runRow(y,iv,{mode,label}){
  const s2=slices(iv);
  const peak=maxN(s2);
  let s=`<text x="2" y="${y+3}" ${MONO} font-size="8.5" fill="${C.mut}">Run</text>`;
  // the ground: elapsed time we were NOT on their compute
  s+=`<rect x="${LBL}" y="${y-2.5}" width="${PLOT}" height="5" rx="1.2" fill="${C.idle}"/>`;
  s2.forEach(sl=>{
    const h = mode==='height' ? 3+ (sl.n/peak)*7 : 8;
    const fill = mode==='flat' ? COMPUTE : sl.ok ? C.good : sl.bad ? C.bad : COMPUTE;
    s+=`<rect x="${px(sl.a).toFixed(1)}" y="${(y-h/2).toFixed(1)}" width="${Math.max(sl.ok?1.4:3.2,((sl.b-sl.a)/100)*PLOT).toFixed(1)}" height="${h.toFixed(1)}" rx="1.2" fill="${fill}"/>`;
  });
  s+=dot(LBL,y,EV.queued);
  s+=dot(LBL+PLOT,y,EV.ok);
  if(label) s+=`<text x="${W-4}" y="${y+3}" ${MONO} font-size="7.5" fill="${C.mut}" text-anchor="end">${label}</text>`;
  return s;
}

const svg=(inner,h,l)=>`<svg viewBox="0 0 ${W} ${h}" role="img" aria-label="${l}">${inner}</svg>`;
const cap=(t,y)=>`<text x="${LBL}" y="${y}" ${MONO} font-size="7" fill="${C.mut}">${t}</text>`;
const out={};

// today vs compute profile
out.idea=svg(
  cap('today: the run’s status, full width, repeating the axis',14)+
  `<text x="2" y="${34+3}" ${MONO} font-size="8.5" fill="${C.mut}">Run</text>`+
  `<rect x="${LBL}" y="${34-4}" width="${PLOT}" height="8" rx="1.2" fill="${C.good}"/>`+
  dot(LBL,34,EV.queued)+dot(LBL+PLOT,34,EV.ok)+
  `<text x="${W-4}" y="${37}" ${MONO} font-size="7.5" fill="${C.mut}" text-anchor="end">2.069s</text>`+
  cap('as an execution profile: grey is elapsed, colour is SDK execution',66)+
  runRow(86,RETRY,{mode:'outcome',label:'19ms compute / 2.069s'}),
  106,'The run row today compared with a compute profile');

// the three treatments, on wide
out.wide=svg(
  cap('flat: one colour, no outcome',14)+runRow(30,WIDE,{mode:'flat',label:'12 steps'})+
  cap('outcome: coloured by what ran in each slice',60)+runRow(76,WIDE,{mode:'outcome',label:'all succeeded'})+
  cap('height: thickness is how many ran at once',106)+runRow(126,WIDE,{mode:'height',label:'peak 12 at once'}),
  146,'Three treatments of the run bar on the wide fixture');

// the ambiguous case
out.mixed=svg(
  cap('gnarly-mixed: one branch threw, one returned, both caught',14)+
  runRow(34,MIXED,{mode:'outcome',label:'138ms compute / 324ms'})+
  `<text x="${px(52.5)}" y="${20}" ${MONO} font-size="6.5" fill="${C.good}" text-anchor="middle">ok</text>`+
  `<text x="${px(55.7)}" y="${50}" ${MONO} font-size="6.5" fill="${C.bad}" text-anchor="middle">failed</text>`+
  cap('almost none of this run was SDK execution, which the old bar hid',60),
  72,'gnarly-mixed drawn as a compute profile');

fs.writeFileSync(HERE+'runbar.json',JSON.stringify(out));
console.log('ok',Object.keys(out).join(' '));
