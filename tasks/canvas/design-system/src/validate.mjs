const HERE=new URL('./',import.meta.url).pathname;
/**
 * Reads the generated figures back and checks each row's event sequence against
 * the rules, so a fragment cannot quietly describe something impossible.
 */
import fs from 'fs';

const GENS=['vocab','barvocab','attribution','items-a','items-bc','items-disc','items-more','connect','batch1','runbar'];
const KIND=[
  [/ev-hollow-bad"/,'retry'],
  [/ev-queued"/,'queued'],
  [/ev-ribbon"/,'planned'],
  [/ev-hollow"/,'started'],
  [/ev-disc"/,'discovery'],
  [/ev-ok"/,'ok'],
  [/ev-failed"/,'failed'],
  [/ev-timeout"/,'timeout'],
];
const classify=c=>{ for(const [re,name] of KIND) if(re.test(c)) return name; return null; };

const OPENERS=new Set(['queued','planned']);
const FINAL=new Set(['ok','failed','timeout','cancelled']);

function rowsOf(svg){
  const out=new Map();
  for(const m of svg.matchAll(/<circle class="ev [^"]*" cx="([\d.]+)" cy="([\d.]+)"[^>]*\/>/g)){
    const [full,x,y]=m;
    const k=classify(full); if(!k) continue;              // background discs
    const key=Math.round(+y);
    if(!out.has(key)) out.set(key,[]);
    out.get(key).push({x:+x,k});
  }
  for(const arr of out.values()) arr.sort((a,b)=>a.x-b.x);
  return out;
}

function check(seq){
  const p=[];
  const k=seq.map(s=>s.k);
  if(!k.length) return p;
  if(!OPENERS.has(k[0])) p.push(`opens on "${k[0]}" — a row must open on queued or planned`);
  for(let i=1;i<k.length;i++) if(k[i]===k[i-1] && Math.abs(seq[i].x-seq[i-1].x)>1)
    p.push(`two "${k[i]}" marks in a row`);
  for(let i=0;i<k.length-1;i++) if(FINAL.has(k[i]))
    p.push(`"${k[i]}" is a resolution but is not last`);
  const cancelled = k.length && !FINAL.has(k[k.length-1]);
  if(k.filter(x=>x==='started').length>1 && !k.includes('retry'))
    p.push('more than one "started" without a retry between them');
  return p;
}

let figures=0, rows=0, bad=0;
for(const g of GENS){
  const J=JSON.parse(fs.readFileSync(HERE+g+'.json','utf8'));
  const svgs=[];
  const walk=v=>{ if(typeof v==='string'){ if(v.startsWith('<svg')) svgs.push(v); }
                  else if(Array.isArray(v)) v.forEach(walk);
                  else if(v&&typeof v==='object') Object.values(v).forEach(walk); };
  walk(J);
  svgs.forEach((svg,i)=>{
    if(/aria-label="[^"]*(kinds|colours mean)/.test(svg)) return;
    figures++;
    for(const [y,seq] of rowsOf(svg)){
      rows++;
      const probs=check(seq);
      if(probs.length){ bad++; console.log(`  ${g} fig#${i} y=${y}: ${seq.map(s=>s.k).join(' -> ')}`);
        probs.forEach(p=>console.log(`      ! ${p}`)); }
    }
  });
}
console.log(`\n${figures} figures, ${rows} rows, ${bad} with problems`);
