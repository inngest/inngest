/**
 * Does the moments -> bars derivation reproduce what the figures author?
 *
 * Every authored row is turned into the moments it implies (the same reading
 * `autoDots` already does to place the circles), those moments are pushed back
 * through `derive`, and the result compared with the bars that were written by
 * hand. A row that round-trips can be rewritten as a list of moments with no
 * change to the drawing. A row that cannot is carrying something the moments do
 * not say, and that is the list worth looking at.
 */
import fs from 'fs';
import * as R from './rules.mjs';
import { base, isFail, discovered, fillGaps, autoDots, EV } from './vocabulary.mjs';

const rows=JSON.parse(fs.readFileSync(new URL('./authored.json',import.meta.url).pathname,'utf8'));

/** What kind of row is this? The substance of its work says so. */
const KIND_OF={disc:'disc', child:'child', wait:'wait', waitok:'wait', waitout:'wait',
  waitstop:'wait', spanok:'span', spanerr:'span', spanunset:'span'};
const WORK=k=>!['idle','backoff','hold'].includes(base(k));
/**
 * What a row is, and whether its head belongs to the request that reported it.
 * A discovery bar followed by work of another substance is that shape: the
 * request is at the head, the step's own work follows.
 */
const shapeOf=segs=>{
  const work=segs.map(([k])=>k).filter(WORK);
  const after=work.filter(k=>base(k)!=='disc');
  const reported=work.some(k=>base(k)==='disc') && after.length>0;
  const decisive=(reported?after:work)[0];
  return {kind:decisive?(KIND_OF[base(decisive)]||'step'):'step', reported};
};

/** The moment that OPENS an interval of not-working. */
const OPENED_BY={idle:'queued', backoff:'retry', hold:'held'};
/** The moment that CLOSES an interval of work, once nothing follows it. */
const CLOSED_BY={good:'ok', disc:'ok', child:'ok', spanok:'ok', waitok:'ok',
  bad:'failed', spanerr:'failed', spanunset:'done', waitout:'timeout',
  stopped:'cancelled', waitstop:'cancelled'};

/**
 * The moments a row of bars implies. Every boundary is a moment: a work bar
 * ends by resolving, a not-working bar ends by something starting.
 */
const momentsOf=segs=>{
  const at=[]; const add=(n,p,s)=>{ if(n && !(at.length && at[at.length-1][0]===n && at[at.length-1][1]===p)) at.push(s?[n,+p.toFixed(4),s]:[n,+p.toFixed(4)]); };
  const lastWork=segs.map(([k])=>WORK(k)).lastIndexOf(true);
  segs.forEach(([k,x],i)=>{
    const prev=segs[i-1];
    if(prev && WORK(prev[0]))
      // the previous work resolved here; another attempt to come makes it a retry
      add(isFail(prev[0]) && i<=lastWork ? 'retry' : CLOSED_BY[base(prev[0])], x);
    add(WORK(k) ? 'started' : OPENED_BY[base(k)], x, WORK(k) ? (KIND_OF[base(k)]||'step') : null);
  });
  const last=segs[segs.length-1];
  if(WORK(last[0]) && !['running','wait'].includes(base(last[0])))
    add(isFail(last[0])&&base(last[0])!=='bad'?'failed':CLOSED_BY[base(last[0])], last[1]+last[2]);
  return at;
};

const norm=s=>s.map(([k,x,w])=>[k,+x.toFixed(2),+w.toFixed(2)]);
const same=(a,b)=>JSON.stringify(norm(a))===JSON.stringify(norm(b));

let ok=0; const bad=[];
for(const r of rows){
  const segs=fillGaps(r.segs.map(([k,x,w])=>({kind:k,x,w}))).map(g=>[g.kind,g.x,g.w]);
  const at=momentsOf(segs);
  const end=segs.length?segs[segs.length-1][1]+segs[segs.length-1][2]:0;
  const sh=shapeOf(segs);
  const at2=at.map((mo,j)=>mo.length===3 && (mo[2]===sh.kind || (sh.reported && j===0)) ? [mo[0],mo[1]] : mo);
  const got=R.derive(sh.kind, at2, end, {reported:sh.reported});
  if(same(segs,got)) ok++;
  else bad.push({n:r.n, kind:(sh.reported?'rep/':'')+sh.kind, at, want:norm(segs), got:norm(got)});
}
console.log(`${ok}/${rows.length} rows round-trip (${(100*ok/rows.length).toFixed(0)}%)`);
const by={};
for(const b of bad){
  const key=b.kind+': '+JSON.stringify(b.want.map(s=>s[0]))+' vs '+JSON.stringify(b.got.map(s=>s[0]));
  (by[key]=by[key]||[]).push(b.n);
}
Object.entries(by).sort((a,b)=>b[1].length-a[1].length).slice(0,14)
  .forEach(([k,v])=>console.log(`  ${String(v.length).padStart(3)}  ${k}`));

if(process.env.DETAIL) bad.forEach(b=>console.log('\n'+b.n+'  ['+b.kind+']\n  at   '+JSON.stringify(b.at)+'\n  want '+JSON.stringify(b.want)+'\n  got  '+JSON.stringify(b.got)));
