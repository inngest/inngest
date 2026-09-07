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
  [/ev-done"/,'done'],
  [/ev-ok"/,'ok'],
  [/ev-failed"/,'failed'],
  [/ev-timeout"/,'timeout'],
];
const classify=c=>{ for(const [re,name] of KIND) if(re.test(c)) return name; return null; };

const OPENERS=new Set(['queued','planned','started']);
const FINAL=new Set(['ok','failed','timeout','cancelled','done']);

function rowsOf(svg){
  const out=new Map();
  for(const m of svg.matchAll(/<circle class="ev [^"]*" cx="([\d.]+)" cy="([\d.]+)"[^>]*\/>/g)){
    const [full,x,y]=m;
    // The trace frame (minimap, Run, Finalization) shares the row grid with the
    // figure it surrounds. It is context, not a row under test.
    if(/class="ev ctx /.test(full)) continue;
    const k=classify(full); if(!k) continue;              // background discs
    const key=Math.round(+y);
    if(!out.has(key)) out.set(key,[]);
    out.get(key).push({x:+x,k});
  }
  // A row is drawn twice when part of it is lit: the whole row at its dim
  // opacity, then the lit parts repainted over it. The repaint is the SAME
  // mark, so a circle at the same position and kind is dropped rather than
  // counted as a second one.
  for(const [k,arr] of out){

    // Dedupe by identity, not against the neighbour: a row can carry two
    // different marks at the same instant (a wait enqueued and started
    // together), and after the repaint those four interleave, so an adjacent
    // comparison drops nothing. Sort is stable, so draw order survives.
    const seen=new Set();
    const uniq=arr.filter(d=>{ const id=d.x.toFixed(2)+"|"+d.k;
      if(seen.has(id)) return false; seen.add(id); return true; });
    uniq.sort((a,b)=>a.x-b.x);
    out.set(k, uniq);
  }
  return out;
}

function check(seq){
  const p=[];
  const k=seq.map(s=>s.k);
  if(!k.length) return p;
  /**
   * A row opens on the moment it was actually in.
   *
   * This used to insist on queued or planned, because marks were read back off
   * the bars and the reader called a row's first mark queued whatever it was.
   * Marks come from the row's moments now, so a row with no queue in front of
   * it opens on `started` and says so -- which is what a captured fixture looks
   * like once the run's leading queue has been trimmed off the front, and what
   * an unresolved sleep looks like at any time.
   */
  if(!OPENERS.has(k[0])) p.push(`opens on "${k[0]}" — a row must open on queued, planned or started`);
  for(let i=1;i<k.length;i++) if(k[i]===k[i-1] && Math.abs(seq[i].x-seq[i-1].x)>1)
    p.push(`two "${k[i]}" marks in a row`);
  for(let i=0;i<k.length-1;i++) if(FINAL.has(k[i]))
    p.push(`"${k[i]}" is a resolution but is not last`);
  // A row may execute more than once, but only for a reason the row shows.
  // Two of them: a `retry` (the attempt threw, another follows) and a `planned`
  // (the discovery request finished and handed the row to the step it reported
  // — a request and the step it planned share one row). A second `started` with
  // neither between them is a row that began executing twice for no stated
  // reason, which is the thing this check exists to catch.
  const REEXEC=new Set(['retry','planned','queued']);
  for(let i=1,last=-1;i<k.length;i++){
    if(k[i]!=='started') continue;
    if(last<0){ last=i; continue; }
    if(!k.slice(last+1,i).some(x=>REEXEC.has(x)))
      p.push('a second "started" with no retry or planned between them');
    last=i;
  }
  return p;
}

/**
 * Every figure's code example must name the steps the figure actually draws.
 *
 * The snippets were first written from each figure's caption rather than its
 * rows, and 34 of 51 named steps the drawing did not have — a step called `a`
 * beside a row called `doomed`. Nothing caught it, because a caption and a
 * drawing can disagree forever without either being wrong on its own.
 *
 * A row label is the step id. Group rows (`× 40 batch`) and request rows
 * (`req + a`) carry an id inside them, so a substring match is the right test.
 */
/**
 * The Run row is a profile of the whole run, so it has to reach the end of the
 * run — which is the end of finalization, not the end of the last user step.
 *
 * It was built from the figure's own rows only, so it stopped short and showed
 * the run doing nothing while finalization executed: the overview disagreeing
 * with the rows beneath it, which is the one thing an overview must never do.
 * Derived now, and asserted here so it stays derived.
 */
/**
 * The generated page has to be a well-formed document.
 *
 * An unterminated CSS declaration in a style ATTRIBUTE once made the figure
 * exporter swallow the whole page into every doc image, which the docs tab then
 * inlined — 16 `</main>` tags, one `<main>`, and a script that never closed, so
 * the page stopped parsing and every tab died. Nothing in the build noticed:
 * the figures were right, the rules were right, the HTML was rubble.
 */
function checkPageStructure(){
  const html=fs.readFileSync(HERE+'../trace-design-system.html','utf8');
  const n=re=>(html.match(re)||[]).length;
  const bad=[];
  const pairs=[['<main>',/<main>/g,/<\/main>/g],['<script>',/<script[ >]/g,/<\/script>/g]];
  for(const [name,open,close] of pairs){
    const a=n(open), b=n(close);
    if(a!==b) bad.push(`${a} ${name} against ${b} closing`);
  }
  // A doc figure is a few KB; a megabyte means the page leaked into it.
  let dir=[];
  try{ dir=fs.readdirSync(HERE+'../../docs/images/'); }catch{}
  for(const f of dir){
    if(!f.endsWith('.svg')) continue;
    const size=fs.statSync(HERE+'../../docs/images/'+f).size;
    if(size>60000) bad.push(`docs/images/${f} is ${(size/1024).toFixed(0)}KB`);
  }
  bad.forEach(b=>console.log('  '+b));
  return bad.length;
}

function checkRunExtent(){
  let bad=0;
  for(const g of ['items-a','items-bc','items-disc','items-more']){
    const J=JSON.parse(fs.readFileSync(HERE+g+'.json','utf8'));
    const svgs=[];
    const walk=v=>{ if(typeof v==='string'){ if(v.startsWith('<svg')) svgs.push(v); }
                    else if(Array.isArray(v)) v.forEach(walk);
                    else if(v&&typeof v==='object') Object.values(v).forEach(walk); };
    walk(J);
    svgs.forEach((svg,i)=>{
      const marks=[...svg.matchAll(/<circle class="ev ctx [^"]*" cx="([\d.]+)" cy="([\d.]+)"/g)]
        .map(m=>({x:+m[1],y:Math.round(+m[2])}));
      if(!marks.length) return;                       // unframed figure
      const byRow={};
      for(const m of marks) (byRow[m.y]=byRow[m.y]||[]).push(m.x);
      const ys=Object.keys(byRow).map(Number).sort((a,b)=>a-b);
      if(ys.length<2) return;                         // figure draws its own Run row
      const runEnd=Math.max(...byRow[ys[0]]);
      const finEnd=Math.max(...byRow[ys[ys.length-1]]);
      if(Math.abs(runEnd-finEnd)>0.5){
        bad++;
        console.log(`  ${g} fig#${i}: Run row ends at ${runEnd.toFixed(1)} but the trace ends at ${finEnd.toFixed(1)}`);
      }
    });
  }
  return bad;
}

async function checkCodeSync(){
  const {CODE}=await import('./code.mjs');
  const ds=fs.readFileSync(HERE+'ds.mjs','utf8');
  const sc=ds.slice(ds.indexOf('const SCEN=['), ds.indexOf('const scenarios ='));

  /**
   * The page groups figures under the code that produced them, so the check is
   * per GROUP: every row a figure draws has to be named by its group's snippet,
   * and every step the snippet names has to be drawn by one of the figures in
   * the group. One snippet standing in for several figures is the whole point
   * of the arrangement; a snippet naming a step none of them draw is a group
   * that has drifted from its code.
   */
  const groups=[];
  for(const g of sc.matchAll(/\['([A-Za-z0-9]+)',\[([\s\S]*?)\n \]\]/g))
    groups.push([g[1], [...g[2].matchAll(/\['([a-z]+\d+[a-z]*)'/g)].map(x=>x[1])]);

  const EX={};
  for(const g of ['items-a','items-bc','items-disc','items-more'])
    Object.assign(EX, JSON.parse(fs.readFileSync(HERE+g+'.json','utf8')));

  const labels=svg=>[...svg.matchAll(/<text x="(?:9|15)" y="[\d.]+"[^>]*>([^<]*)<\/text>/g)]
    .map(m=>m[1]).filter(r=>r && !/^(Run|Finalization)$/.test(r));
  const ids=code=>[...code.matchAll(/step\.(?:run|sleep|waitForEvent|waitForSignal|invoke|sendEvent)\(\s*['"\`]([^'"\`]+)/g)]
    .map(m=>m[1]).filter(n=>!n.includes('${'));

  // One figure legitimately names a step the drawing does not contain: the
  // request that would have reported `b` never succeeded, so no row for it
  // exists. That absence IS the figure. Exemptions carry their reason.
  const EXEMPT={ c68:'the step its code names was never created — that is the point' };
  let bad=0;
  const seen=new Set();
  for(const [k,figs] of groups){
    const code=CODE[k];
    if(!code){ console.log(`  ${k}: a group with no code example`); bad++; continue; }
    const anyName=[...code.matchAll(/['"\`]([^'"\`\n]+)['"\`]/g)].map(m=>m[1])
      .filter(n=>!n.includes('${'));
    const bare=r=>r.replace(/^[\s↳·]+/,'').trim();
    let drawnAll=[];
    for(const id of figs){
      seen.add(id);
      const e=EX[id];
      if(!e){ console.log(`  ${id}: named by group ${k} but no such figure`); bad++; continue; }
      const drawn=labels(e.frames[0].svg);
      drawnAll=drawnAll.concat(drawn);
      if(EXEMPT[id]) continue;
      const orphanRows=drawn.filter(r=>{
        const t=bare(r);
        return !anyName.some(n=>t.includes(n)||n.includes(t));
      });
      if(orphanRows.length){
        bad++;
        console.log(`  ${id} (group ${k}): rows the code never names [${orphanRows.join(', ')}]`);
      }
    }
    // A group carrying an exempt figure inherits its exemption: c68's snippet
    // names the step no request ever managed to create, which is the figure.
    const exemptGroup=figs.some(id=>EXEMPT[id]);
    const orphanSteps=exemptGroup?[]:ids(code).filter(n=>!drawnAll.some(r=>r.includes(n)));
    if(orphanSteps.length){
      bad++;
      console.log(`  ${k}: steps no figure in the group draws [${orphanSteps.join(', ')}]`);
    }
  }
  for(const k of Object.keys(CODE)) if(!groups.some(([g])=>g===k)){
    console.log(`  ${k}: code example for a group that no longer renders`); bad++;
  }
  return bad;
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

let unruled=[]; try{ unruled=JSON.parse(fs.readFileSync(HERE+'unruled.json','utf8')); }catch{}
if(unruled.length){
  console.log(`\n${unruled.length} figure(s) hand-place geometry the elastic rule cannot reach:`);
  unruled.sort((a,b)=>b.worst-a.worst).slice(0,5)
    .forEach(x=>console.log(`  ${String(x.worst).padStart(3)}% idle  ${x.label}`));
  console.log('  (a figure should be a list of events; the drawing follows from the rules)');
}
// The Run row is derived, never authored. A figure that draws its own is a
// second account of the same run, kept in step by hand until it is not.
const authored=(()=>{
  let n=0;
  for(const f of ['items-a.mjs','items-bc.mjs','items-disc.mjs','items-more.mjs','fixtures.mjs',
                  'batch1.mjs','runbar.mjs','annotated.mjs','attribution.mjs']){
    let src=''; try{ src=fs.readFileSync(HERE+f,'utf8'); }catch{ continue; }
    const hits=(src.match(/\{n:'Run'/g)||[]).length;
    if(hits){ console.log(`  ${f} draws ${hits} Run row(s) by hand`); n+=hits; }
  }
  return n;
})();
if(authored) console.log(`\n${authored} hand-authored Run row(s) — the Run row is derived from the trace`);
const broken=checkPageStructure();
if(broken) console.log(`\n${broken} structural problem(s) in the generated page`);
const short=checkRunExtent();
if(short) console.log(`\n${short} figure(s) whose Run row does not reach the end of the trace`);
const desync=await checkCodeSync();
console.log(desync
  ? `\n${desync} figure(s) whose code and drawing disagree`
  : 'code and figures agree');
if(bad||desync||short||broken||authored) process.exitCode=1;
