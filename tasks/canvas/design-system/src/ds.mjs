const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import { execFileSync } from 'child_process';
import * as V from './vocabulary.mjs';
const GENS=['vocab','barvocab','attribution','items-a','items-bc','items-disc','items-more','connect','batch1','runbar','annotated','steptypes','detail','primer','fixtures'];
for(const g of GENS) execFileSync('node',[HERE+g+'.mjs'],{stdio:'pipe'});
const J=Object.fromEntries(GENS.map(g=>[g,JSON.parse(fs.readFileSync(HERE+g+'.json','utf8'))]));
const EX={...J['items-a'],...J['items-bc'],...J['items-disc'],...J['items-more']};
/**
 * The docs tab shows the EXPORTED figures, not the live framed ones, because
 * the point of the tab is to be the page that ships. The exports are unframed;
 * the blurred surround is a review device and would not appear in user docs.
 * A stale image here means `node export-docs.mjs` has not been re-run, which is
 * exactly the drift the tab exists to make visible.
 */
const DOCFIG=(()=>{
  const dir=HERE+'../../docs/images/', out={};
  let names=[]; try{ names=fs.readdirSync(dir); }catch{ return out; }
  for(const f of names) if(f.endsWith('.svg'))
    out[f.replace(/\.svg$/,'')]=fs.readFileSync(dir+f,'utf8').replace(/^<\?xml[^>]*>/,'');
  return out;
})();

/**
 * Margin notes with leaders that land on an exact point in the plot.
 *
 * One overlay SVG sits over the figure in rendered-pixel coordinates and is
 * allowed to draw outside it, so a leader can start beside the figure and end
 * on a specific bar. Nothing here participates in layout.
 */
const PAD_X=8, PAD_Y=16, FIG=960;
const annofig=(o)=>{
  const g=o.geomShared, S=(FIG-PAD_X*2)/g.W;
  const X=p=>PAD_X+(g.LBL+(p/100)*g.PLOT)*S;
  const Y=r=>PAD_Y+(g.TOP+9+g.ROW*r)*S;
  const H=PAD_Y*2+g.ROW*4*S;
  const wob=(x1,y1,x2,y2,seed)=>{
    // two control points, pushed by a per-note seed, so no two leaders share a curve
    const r=(n)=>((Math.sin(seed*97.3+n*41.7)*43758.5453)%1);
    const dx=x2-x1, dy=y2-y1;
    const bow=(0.18+Math.abs(r(1))*0.30)*(r(2)>0?1:-1);
    const c1x=x1+dx*0.30, c1y=y1+dy*0.30+dx*bow*0.5;
    const c2x=x1+dx*0.72, c2y=y1+dy*0.78-dx*bow*0.35;
    return `M${x1.toFixed(1)} ${y1.toFixed(1)} C${c1x.toFixed(1)} ${c1y.toFixed(1)} ${c2x.toFixed(1)} ${c2y.toFixed(1)} ${x2.toFixed(1)} ${y2.toFixed(1)}`;
  };
  const head=(x,y,dir,seed)=>{
    const t=((Math.sin(seed*13.7)*1000)%1)*0.5-0.25;
    const L=11, a1=t+0.55, a2=t-0.55;
    const p=(a)=>`${(x-L*Math.cos(a)*dir).toFixed(1)} ${(y-L*Math.sin(a)).toFixed(1)}`;
    return `<path d="M${p(a1)} L${x.toFixed(1)} ${y.toFixed(1)} L${p(a2)}" fill="none" stroke="var(--pen)" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>`;
  };

  let leaders='', labels='';
  o.notes.forEach((n,ni)=>{
    const y=Y(n.row);
    const right=n.side==='right';
    const startX = right ? FIG+14 : -14;
    let tx, ty=y, extra='';
    if(n.span){
      const [a,b]=n.span.map(X);
      // a bracket under the range, with the leader meeting its middle
      extra=`<path d="M${a.toFixed(1)} ${(y+9).toFixed(1)} L${a.toFixed(1)} ${(y+13).toFixed(1)} L${b.toFixed(1)} ${(y+13).toFixed(1)} L${b.toFixed(1)} ${(y+9).toFixed(1)}" fill="none" stroke="var(--pen)" stroke-width="1.2" stroke-linecap="round" opacity=".9"/>`;
      tx=(a+b)/2; ty=y+15;
    } else {
      tx=X(n.at)+(right?9:-9);
    }
    const dir = right ? -1 : 1;
    leaders += `<g filter="url(#pen-shadow)"><path d="${wob(startX,y,tx,ty,ni+1)}" fill="none" stroke="var(--pen)" stroke-width="1.6" stroke-linecap="round" opacity=".95"/>`+head(tx,ty,dir,ni+1)+`</g>`+extra;
    labels += `<div class="anno ${n.side}" style="top:${(y-11).toFixed(0)}px">${n.text}</div>`;
  });

  return `<figure class="annofig"><div class="figure">${o.svg}</div>`+
    `<svg class="leads" viewBox="0 0 ${FIG} ${H.toFixed(0)}" width="${FIG}" height="${H.toFixed(0)}" aria-hidden="true"><defs><filter id="pen-shadow" x="-20%" y="-40%" width="140%" height="220%"><feDropShadow dx="0" dy="1.6" stdDeviation="1.4" flood-color="#000" flood-opacity=".65"/></filter></defs>${leaders}</svg>`+
    labels+`</figure>`;
};
import {CODE} from './code.mjs';
const esc=t=>t.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');

/**
 * Enough colouring to read the shape of a snippet: what is a step call, what is
 * a name, what is an aside. Deliberately not a real tokeniser — these are eight
 * lines of pseudo-code, and a full grammar would be more machinery than the
 * thing it colours.
 *
 * Order matters: comments and strings are taken first and stashed, so a keyword
 * inside a comment is not coloured as a keyword.
 */
const KW=/\b(await|const|async|return|for|of|try|catch|export|default|new|throw|if)\b/g;
function highlight(src){
  const held=[];
  const hold=t=>{held.push(t);return '\u0000'+(held.length-1)+'\u0000';};
  let t=esc(src)
    .replace(/\/\/[^\n]*/g, m=>hold('<i class="c-com">'+m+'</i>'))
    .replace(/&#39;[^&]*?&#39;|'[^']*'|`[^`]*`/g, m=>hold('<i class="c-str">'+m+'</i>'));
  t=t.replace(KW,'<i class="c-kw">$1</i>')
     .replace(/\b(step|Promise|inngest)\b/g,'<i class="c-obj">$1</i>')
     .replace(/\.(run|sleep|waitForEvent|waitForSignal|invoke|sendEvent|all|race|map|createFunction)\b/g,'.<i class="c-fn">$1</i>')
     .replace(/\u2026/g,'<i class="c-dim">\u2026</i>');
  return t.replace(/\u0000(\d+)\u0000/g,(m,i)=>held[+i]);
}
const codeOf=id=>CODE[id]?`<pre class="code">${highlight(CODE[id])}</pre>`:'';
/**
 * The user-facing docs, rendered from the same markdown that ships, with its
 * figures resolved back to the live SVGs rather than the exported files. That
 * is the point of having it here: if a rule changes, the docs tab shows it
 * immediately and you can see whether the prose still tells the truth.
 *
 * Small on purpose — enough markdown for this one document, not a parser.
 */
const DOC=(()=>{
  let md;
  try{ md=fs.readFileSync(HERE+'../../docs/understanding-traces.md','utf8'); }
  catch{ return '<p class="note">docs/understanding-traces.md not found</p>'; }
  md=md.replace(/^---[\s\S]*?^---\n/m,'');                       // front matter
  const imgs={};
  md=md.replace(/!\[([^\]]*)\]\(\.\/images\/([a-z-]+)\.svg\)/g,(m,alt,name)=>{
    imgs[name]=alt; return '\u0001'+name+'\u0001';
  });
  const inline=t=>esc(t)
    .replace(/`([^`]+)`/g,'<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g,'<strong>$1</strong>')
    .replace(/(?<![*\w])\*([^*]+)\*(?!\*)/g,'<em>$1</em>');
  const out=[];
  for(const block of md.split(/\n{2,}/)){
    const b=block.trim(); if(!b) continue;
    if(b.startsWith('\u0001')){
      const name=b.replace(/\u0001/g,'');
      out.push(DOCFIG[name]
        ? `<figure class="frame"><div class="figure">${DOCFIG[name]}</div><figcaption class="fl">${esc(imgs[name]||'')}</figcaption></figure>`
        : `<p class="note">missing figure: ${name}</p>`);
    }
    else if(b.startsWith('```')) out.push(`<pre class="code">${highlight(b.replace(/^```\w*\n?|\n?```$/g,''))}</pre>`);
    else if(b.startsWith('## ')) out.push(`<h3>${inline(b.slice(3))}</h3>`);
    else if(b.startsWith('### ')) out.push(`<h4>${inline(b.slice(4))}</h4>`);
    else if(b.startsWith('> ')) out.push(`<p class="rule">${inline(b.replace(/^> ?/gm,''))}</p>`);
    else if(/^\|/.test(b)){
      const rows=b.split('\n').filter(r=>!/^\|[\s|:-]+\|$/.test(r))
        .map(r=>r.split('|').slice(1,-1).map(c=>`<td>${inline(c.trim())}</td>`).join(''));
      out.push(`<table>${rows.map(r=>`<tr>${r}</tr>`).join('')}</table>`);
    }
    else if(/^[-*] /m.test(b)) out.push(`<ul>${b.split(/\n(?=[-*] )/).map(li=>`<li>${inline(li.replace(/^[-*] /,'').replace(/\n\s+/g,' '))}</li>`).join('')}</ul>`);
    else if(/^\d+\. /.test(b)) out.push(`<ol>${b.split(/\n(?=\d+\. )/).map(li=>`<li>${inline(li.replace(/^\d+\. /,'').replace(/\n\s+/g,' '))}</li>`).join('')}</ol>`);
    else out.push(`<p>${inline(b.replace(/\n/g,' '))}</p>`);
  }
  return out.join('\n');
})();

const chip=k=>`<svg class="chip" viewBox="0 0 30 8" aria-hidden="true"><rect width="30" height="8" rx="1.5" fill="${V.paint(k)}"/></svg>`;
const mark=k=>`<svg class="chip mk" viewBox="0 0 12 12" aria-hidden="true">${V.dot(6,6,k,1,3.6,false)}</svg>`;
const keys=rows=>`<ul class="keylist">`+rows.map(([sw,t])=>`<li>${sw}<span>${t}</span></li>`).join('')+`</ul>`;
const fig=(svg,cap='',cls='')=>`<figure><div class="figure ${cls}">${svg}</div>${cap?`<figcaption>${cap}</figcaption>`:''}</figure>`;
const frames=id=>{const e=EX[id];if(!e)throw new Error('no '+id);
  return `<div class="frames${e.frames.length>1?' two':''}">`+
    e.frames.map(f=>`<figure class="frame"><div class="figure">${f.svg}</div><figcaption class="fl">${f.l}</figcaption></figure>`).join('')+`</div>`;};

/** id -> one line saying what the scenario shows. */
const SC=[
 ['Sequential', [
  ['c0','One request, one step. The SDK reports and executes in the same request, so there is no discovery bar.'],
 ]],
 ['Fan-out and coalesce', [
  ['c1','One request reports three steps. The ribbon threads their enqueue marks, which are blue because a discovery request put them there.'],
  ['c2','Three steps resolve and cause the next request. That request reported one step, so it rolls into that step’s row.'],
  ['c4','The ribbon covers exactly the steps the request reported, so an uneven fan-out is countable without interacting. Nothing static predicted that width: the ribbon reports what happened and promises no shape.'],
  ['c3','Fan-out inside a fan-out. One ribbon per discovery request.'],
  ['c6','One resumption reports two steps. Both sit under one ribbon, so no order is implied between them.'],
  ['c8','Two branches that each caused their own request. Two requests, no ribbon.'],
 ]],
 ['Attribution', [
  ['c9','Promise.race(). The winner is a dependency of the next request. The loser resolved after that request started, so it has no cable.'],
  ['c10','A step started but never awaited. No outgoing cable.'],
  ['c11','An SDK that reports one step per response never says these were planned together, so the grouping is recovered from the fact that they overlap. That is a guess, and it is drawn as one: dashed ribbon, dashed cable. Newer SDKs report whole batches and draw solid.'],
 ]],
 ['Platform time', [
  ['c14','Queue time. The step waiting for the executor to pick it up.'],
  ['c15','A discovery request is a separate execution. It happens at the start of a run and on fan-out.'],
  ['c16','Held by flow control. Amber hatched, still not SDK execution time.'],
  ['c16b','Throttle, rate limit and debounce are the same fact and take the same treatment.'],
  ['c17','System latency is queue time and is labelled the same way.'],
  ['c18','Finalization keeps its own row and is not SDK execution time.'],
  ['c19','Everything Inngest did is hatched. Only SDK execution is solid.'],
  ['c20','Every millisecond between a row’s start and its resolution belongs to a named interval.'],
  ['c31','step.sendEvent() is the one row that points out of the run. The ring is hollow because those runs are not in this trace, and the count is the way into them.'],
  ['c21','The gap between a step resolving and the next request starting is drawn once, as the cable and as the interval.'],
 ]],
 ['Outcomes', [
  ['c22','step.run() returned.'],
  ['c23','step.run() threw on the final attempt and the run failed.'],
  ['c24','The function caught the error. Red step, green run.'],
  ['c25','Threw, backed off, retried, returned.'],
  ['c30','Every attempt threw. The only place a circle is filled red.'],
  ['c26','One red row does not tint the rows around it.'],
  ['c29','Still executing. Blue, and no resolution mark: the missing mark is what says it is unresolved.'],
  ['c27','Cancelled while executing.'],
  ['c28','A step.waitForEvent() still open when the run was cancelled.'],
 ]],
  ['Waiting', [
  ['w1','A wait that matched. Blue while it is open, green at the mark where the event arrived.'],
  ['w2','A wait that expired with no match. The run carries on: a timeout is a result the function can act on.'],
  ['w3','A wait inside a fan-out holds the level open. The next request cannot start until the slowest member resolves.'],
  ['w4','Waiting and failing must not read alike. One is hatched and never red; the other is solid and red.'],
 ]],
 ['Time and the axis', [
  ['t1','Dead time is worth almost none of the width. The threshold is low, the band is fixed, and only the drawing compresses — every duration is still wall clock.'],
  ['t1b','The width left over splits by real duration, so a second is the same number of pixels wherever it lands.'],
  ['t1c','Many compressions share one budget. Each band thins rather than the trace spending its width on nothing.'],
  ['t1d','A wait that has not resolved still compresses. No closing mark, and no Finalization row: the run is still going.'],
  ['t1e','A gap in one row is not dead time. Only a stretch with nothing running anywhere is a candidate.'],
  ['t2','Seven days elapsed, 62ms executing. The fill rule reports that before you have read a number.'],
  ['t3','Two tiers of axis label, so a run measured in days keeps its resolution without a second axis.'],
  ['t4','A step too short to draw is still drawn, at a minimum width, with its real duration beside it.'],
 ]],
 ['Scale', [
  ['s1','Five hundred sequential steps collapse to one row that reports the count and draws where each member ran.'],
  ['s2','A wide fan-out collapses the same way. The stagger of a concurrency limit is visible without expanding it.'],
  ['s3','A failure cluster two thirds through a long run is a red smear on the Run row, findable without interacting.'],
  ['s4','About forty rows at rest whatever the step count, and any group expands where you are standing.'],
 ]],
 ['Naming and identity', [
  ['n1','The same step name on two branches. The name is not the identity; the request that reported it is.'],
  ['n2','The SDK’s :1 and :2 suffixes are not stable under parallelism, so nothing in the drawing depends on them.'],
 ]],
 ['Interaction', [
  ['i1','Three tiers of attention: the row, what caused it, everything else. Nothing is removed, only quietened.'],
  ['i2','The Run row is the overview. One picture of the run, not two.'],
 ]],
 ['Honesty', [
  ['h1','Where the trace cannot say what caused a request, it declines rather than guessing.'],
  ['h2','A row’s label and its drawing have to agree about which interval the number names.'],
  ['h3','Nothing is drawn that cannot be asked what it is. Hover any row for its parts.'],
 ]],
 ['Discovery requests that fail', [
  ['c67','The discovery request that would report the next step threw twice before it succeeded.'],
  ['c68','It never succeeded, so no step exists to host it.'],
 ]],
 ['Across runs', [
  ['c51x','']].filter(x=>0)],
];

const scenarios = SC.filter(([,items])=>items.length).map(([title,items])=>
`  <section class="sc">
    <h3>${title}</h3>
    <div class="grid">
${items.map(([id,note])=>`      <div class="item" data-sc="${id}"><div class="col-l">${codeOf(id)}<p class="note">${note}</p></div><div class="col-r">${frames(id)}</div></div>`).join('\n')}
    </div>
  </section>`).join('\n');

const FIX=[['simple','No steps.'],['step','Three steps around a step.sleep().'],
 ['v4sequential','Sequential on an SDK that can report batches. No ribbon anywhere.'],
 ['emit','step.sendEvent() whose events started other runs.'],
 ['invoke','step.invoke(), collapsed and expanded.']];
const fixtures=FIX.map(([id,note])=>
`    <div class="item"><p class="note"><code>${id}</code> ${note}</p>`+
  J.batch1[id].map(p=>fig(p.svg,p.cap)).join('')+`</div>`).join('\n');

/**
 * The configurator.
 *
 * Everything drawable is already indirected through a CSS variable, so the
 * panel never regenerates a figure: it sets variables on :root and all 71
 * SVGs follow. Choices are presets, not free values: the point is to try the
 * system in a different key, not to invent a colour per bar.
 */
/**
 * Live geometry. Every one of these is a CSS custom property the figures read
 * through geometry properties (`height`, `r`) and per-row transforms, so the
 * drawing moves without regenerating — which is the only way to feel what a
 * density actually reads like rather than arguing about a number.
 *
 * All of them go to zero on purpose. A dimension you cannot take away is one
 * you have not tested.
 */
const SLIDERS = [
  {k:'--geo-bar',  n:'step bar height',    d:7,  lo:0, hi:16},
  {k:'--geo-row',  n:'row pitch',          d:17, lo:0, hi:40},
  {k:'--geo-mark', n:'event radius',       d:3,  lo:0, hi:8, st:0.25},
  {k:'--geo-sbar', n:'span bar height',    d:4.2,lo:0, hi:12, st:0.2},
  {k:'--geo-span', n:'span row pitch',     d:9,  lo:0, hi:24},
  {k:'--geo-run',  n:'run row height',     d:8,  lo:0, hi:18},
];

const PALETTES = {
  default:  {good:'#56c295',warn:'#e08770',disc:'#4a72b0',child:'#7d6cb5',hold:'#c2a05a',queued:'#3c4655',muted:'#8593a5','rule-2':'#333d4a',accent:'#7ba3f0'},
  cool:     {good:'#4fb8b0',warn:'#d97b8f',disc:'#5d7fd6',child:'#8f7ad1',hold:'#b9a25f',queued:'#39414f',muted:'#8b97a8','rule-2':'#2f3946',accent:'#6fa8e8'},
  warm:     {good:'#8db860',warn:'#dd6f56',disc:'#c07a3c',child:'#b3688f',hold:'#d3a94e',queued:'#43403c',muted:'#9c9184','rule-2':'#3e3934',accent:'#e0a35c'},
  contrast: {good:'#3ddc9a',warn:'#ff7a5c',disc:'#5b8def',child:'#a07bf0',hold:'#e8b93f',queued:'#4a5568',muted:'#a3b0c2','rule-2':'#3d4855',accent:'#7ba3f0'},
};
/** The hues a kind may be moved to: the palette's own, nothing invented. */
const SWATCHES = ['good','warn','disc','child','hold','muted','rule-2','queued'];
/** bg / line opacity pairs. Solid and hatched are the same mechanism. */
const STYLES = {solid:[1,0], hatch:[.16,.55], dense:[.30,1], faint:[.12,.42]};
const WEIGHTS = {thin:1.2, mid:1.8, thick:2.6};

const swatchRow=(prop,cur)=>SWATCHES.map(s=>
  `<button class="sw${cur===s?' on':''}" data-set="${prop}" data-val="var(--${s})" data-tag="${s}" title="${s}" style="background:var(--${s})"></button>`).join('');

const barRow=([k,[name,desc]])=>`
    <div class="vrow" data-kind="${k}">
      <div class="vhead">
        <svg class="vsw" viewBox="0 0 44 8" aria-hidden="true"><rect width="44" height="8" rx="1.5" fill="${V.paint(k)}"/></svg>
        <b>${name}</b><code>${k}</code>
        <span>${desc}</span>
      </div>
      <div class="ctl">
        <div class="sws">${swatchRow('--bar-'+k,'')}</div>
        <div class="pills">${Object.keys(STYLES).map(s=>
          `<button class="pill${V.SUBSTANCE[k].bg===STYLES[s][0]&&V.SUBSTANCE[k].line===STYLES[s][1]?' on':''}" data-style="${k}" data-val="${s}">${s}</button>`).join('')}</div>
      </div>
    </div>`;

const evRow=([k,[name,desc]])=>`
    <div class="vrow" data-kind="${k}">
      <div class="vhead">
        <svg class="vsw ev" viewBox="0 0 12 12" aria-hidden="true">${V.dot(6,6,k,1,3.4,false)}</svg>
        <b>${name}</b><code>${k}</code>
        <span>${desc}</span>
      </div>
      <div class="ctl">
        <div class="sws">${swatchRow('--ev-'+k,'')}</div>
        <div class="pills">${Object.keys(WEIGHTS).map(w=>
          `<button class="pill" data-weight="${k}" data-val="${w}">${w}</button>`).join('')}</div>
      </div>
    </div>`;

const sidebar=`<aside id="side">
  <svg width="0" height="0" aria-hidden="true" style="position:absolute">${V.HATCH}</svg>
  <div class="sh">
    <button id="fold" title="Collapse the panel" aria-label="Collapse the panel">‹</button>
    <strong>Vocabulary</strong>
    <button id="reset" title="Back to the proposed defaults">reset</button>
  </div>
  <div class="pal">
    <label>Palette</label>
    <div class="pills">${Object.keys(PALETTES).map((p,i)=>
      `<button class="pill${i?'':' on'}" data-pal="${p}">${p}</button>`).join('')}</div>
  </div>
  <h5>Spacing <em>every dimension, down to nothing</em></h5>
  ${SLIDERS.map(sl=>`<div class="sl"><label>${sl.n}<b data-out="${sl.k}">${sl.d}</b></label>
    <input type="range" data-geo="${sl.k}" min="${sl.lo}" max="${sl.hi}" step="${sl.st||0.5}" value="${sl.d}"></div>`).join('')}
  <h5>Bars <em>colour = kind of work; fill = SDK executing</em></h5>
  ${Object.entries(V.BAR_INFO).map(barRow).join('')}
  <h5>Events <em>colour = kind of moment; hollow = not resolved</em></h5>
  ${Object.entries(V.EVENT_INFO).map(evRow).join('')}
  <p class="foot">Hollow and filled is a rule, not a preference. It reports whether
  the row has resolved, so it is not adjustable.</p>
</aside>`;

const scrubs=J.fixtures.map(f=>
`  <div class="item">
    <p class="note"><code>${f.id}</code> ${f.note}</p>
    <div class="scrub" data-fx="${f.id}">
      <div class="figure frame"></div>
      <div class="scrubbar">
        <div class="rail">
          ${f.breaks.map(b=>`<i class="brk" style="left:${b.p0}%;width:${(b.p1-b.p0)}%"></i>`).join('')}
          ${f.events.map(e=>`<button class="tick" data-p="${e.p}"
            style="left:${e.p}%" title="${e.text}"></button>`).join('')}
          <i class="head" style="left:100%"></i>
        </div>
        <input type="range" min="0" max="10000" value="10000" step="1"
               aria-label="Scrub ${f.id} through elapsed time">
      </div>
      <p class="say"><span class="at">100%</span><span class="evt">complete</span></p>
    </div>
  </div>`).join('\n');

const page=`<title>Trace Design System</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Familjen+Grotesk:wght@400;500;600;700&family=Newsreader:opsz,wght@6..72,400&family=JetBrains+Mono:wght@400;500&family=Caveat:wght@500;600&display=swap">
<style>
  :root{color-scheme:dark;
    --ground:#0d1016;--surface:#151a22;--surface-2:#1d242e;
    --ink:#e8edf4;--ink-2:#b8c3d1;--muted:#8593a5;--rule:#242c37;--rule-2:#333d4a;
    --accent:#7ba3f0;--good:#56c295;--warn:#e08770;
    --disc:#4a72b0;--child:#7d6cb5;--queued:#3c4655;--hold:#c2a05a;--pen:#d9c9a8;
    --figw:1240px;
    --display:"Familjen Grotesk","Helvetica Neue",Arial,sans-serif;
    --body:"Newsreader",Georgia,serif;--mono:"JetBrains Mono",ui-monospace,Menlo,monospace;}
  ${V.substanceCSS()}
  ${V.eventCSS()}
  *{box-sizing:border-box}
  body{background:var(--ground);color:var(--ink);font-family:var(--body);font-size:16px;line-height:1.55;margin:0;padding:0;
    display:grid;grid-template-columns:340px minmax(0,1fr);transition:grid-template-columns .18s ease}
  body.folded{grid-template-columns:44px minmax(0,1fr)}
  main{min-width:0;padding:0 32px 80px}

  /* The vocabulary panel: always open, always the reference for what is on
     the right, and the only place any of it is defined. */
  #side{position:sticky;top:0;height:100vh;overflow-y:auto;padding:18px 16px 40px;
    background:var(--surface);border-right:1px solid var(--rule)}
  body.folded #side>*:not(.sh){display:none}
  body.folded #side{padding:14px 6px}
  body.folded #side .sh{flex-direction:column;gap:10px;align-items:center}
  body.folded #side .sh strong,body.folded #side .sh #reset{display:none}
  #fold{padding:2px 6px!important;line-height:1}
  #side .sh{display:flex;align-items:baseline;justify-content:space-between;gap:8px;margin-bottom:14px}
  #side .sh strong{font-family:var(--display);font-size:15px}
  #side button{font-family:var(--mono);font-size:10px;color:var(--muted);background:none;
    border:1px solid var(--rule-2);border-radius:4px;padding:2px 7px;cursor:pointer}
  #side button:hover{color:var(--ink);border-color:var(--muted)}
  #side h5{font-family:var(--display);font-size:12px;text-transform:uppercase;letter-spacing:.09em;
    color:var(--muted);margin:22px 0 8px;font-weight:600;display:flex;flex-direction:column;gap:2px}
  #side h5 em{font-family:var(--body);font-style:normal;text-transform:none;letter-spacing:0;
    font-size:12px;color:var(--rule-2);filter:brightness(1.7)}
  .pal{display:flex;flex-direction:column;gap:5px}
  .pal label{font-family:var(--mono);font-size:9.5px;letter-spacing:.1em;text-transform:uppercase;color:var(--muted)}
  .vrow{padding:6px 0;border-top:1px solid var(--rule)}
  .vhead{display:grid;grid-template-columns:46px auto 1fr;align-items:center;gap:4px 7px;cursor:pointer}
  .vhead b{font-family:var(--display);font-size:12.5px;font-weight:600}
  .vhead code{font-family:var(--mono);font-size:9.5px;color:var(--muted);background:none;padding:0;justify-self:start}
  .vhead span{grid-column:2/4;font-size:12px;color:var(--muted);line-height:1.3}
  .vsw{width:44px;height:8px;display:block}
  .vsw.ev{width:13px;height:13px;margin-left:15px}
  .ctl{display:none;gap:6px;flex-direction:column;padding:8px 0 4px 53px}
  .vrow.open .ctl{display:flex}
  .sl{padding:5px 0 7px}
  .sl label{display:flex;justify-content:space-between;align-items:baseline;
    font-family:var(--display);font-size:11.5px;color:var(--muted);margin-bottom:3px}
  .sl label b{font-family:var(--mono);font-size:10.5px;color:var(--ink-2);font-weight:400}
  .sl input{width:100%;height:14px;-webkit-appearance:none;appearance:none;background:none;cursor:pointer}
  .sl input::-webkit-slider-runnable-track{height:2px;background:var(--rule-2);border-radius:1px}
  .sl input::-moz-range-track{height:2px;background:var(--rule-2);border-radius:1px}
  .sl input::-webkit-slider-thumb{-webkit-appearance:none;width:11px;height:11px;border-radius:50%;
    background:var(--ink-2);margin-top:-4.5px}
  .sl input::-moz-range-thumb{width:11px;height:11px;border:0;border-radius:50%;background:var(--ink-2)}

  /* Geometry, live. height and r are CSS geometry properties, so a bar can take
     its height from a variable; the transform re-centres it on the row's line,
     which the SVG now uses as the bar's y. */
  .figure svg{overflow:visible}
  svg rect.bar{height:var(--geo-bar,7px);transform:translateY(calc(var(--geo-bar,7px) / -2))}
  svg rect.bar.sm{height:var(--geo-sbar,4.2px);transform:translateY(calc(var(--geo-sbar,4.2px) / -2))}
  svg rect.bar.rail{height:1.8px;transform:translateY(-0.9px)}
  svg circle.ev{r:var(--geo-mark,3px)}
  svg circle.ev-bg{r:calc(var(--geo-mark,3px) + 1.3px)}
  /* Each row knows how many gaps of each pitch sit above it. */
  svg g.r{transform:translateY(calc((var(--geo-row,17px) - 17px) * var(--i,0)
                                 + (var(--geo-span,9px) - 9px) * var(--s,0)))}
  .sws{display:flex;gap:4px}
  .sw{width:16px;height:16px;border-radius:3px;border:1px solid var(--rule-2)!important;padding:0!important;cursor:pointer}
  .sw.on{outline:1.5px solid var(--ink);outline-offset:1px}
  .pills{display:flex;gap:4px;flex-wrap:wrap}
  .pill{text-transform:lowercase}
  .pill.on{color:var(--ink)!important;border-color:var(--accent)!important;background:var(--surface-2)!important}
  #side .foot{font-size:11.5px;color:var(--muted);margin:18px 0 0;line-height:1.4}
  @media (max-width:1080px){body{grid-template-columns:1fr}#side{position:static;height:auto;border-right:0;border-bottom:1px solid var(--rule)}}

  /* Scenarios tile: the figures keep their size, the page fits more per row. */
  .grid{display:grid;gap:18px 26px;grid-template-columns:repeat(auto-fill,minmax(min(100%,var(--figw)),1fr))}
  .doc{max-width:74ch}
  .doc h3{margin:34px 0 10px}
  .doc h4{margin:22px 0 8px;font-family:var(--display);font-size:15px}
  .doc figure.frame{margin:16px 0}
  .doc table{margin:14px 0}
  .doc td{padding:6px 14px 6px 0;vertical-align:top;border-top:1px solid var(--rule)}
  .doc li{margin:0 0 7px}
  .doc .code{max-width:60ch}
  .grid .item{border-top:1px solid var(--rule);padding:16px 0 10px;min-width:0;
    display:grid;grid-template-columns:minmax(280px,360px) 1fr;gap:28px;align-items:start}
  /* The code and its explanation stay with you while the figures scroll past.
     Offset by the tab bar, which is sticky at the top of the page. */
  .col-l{position:sticky;top:64px;min-width:0}
  .col-r{min-width:0;display:grid;gap:10px}
  .col-l .note{margin:10px 0 0}
  @media (max-width:1080px){
    .grid .item{grid-template-columns:1fr;gap:12px}
    .col-l{position:static}
  }
  .c-kw{color:var(--accent);font-style:normal}
  .c-obj{color:var(--child);font-style:normal}
  .c-fn{color:var(--good);font-style:normal}
  .c-str{color:var(--hold);font-style:normal}
  .c-com{color:var(--muted);font-style:normal}
  .c-dim{color:var(--muted);font-style:normal}
  .code{font-family:var(--mono,ui-monospace,monospace);font-size:13px;line-height:1.6;color:var(--ink-2);
    background:var(--surface-2,rgba(255,255,255,.03));border-left:2px solid var(--rule-2);
    padding:10px 12px;margin:0 0 10px;white-space:pre-wrap;overflow-wrap:break-word;
    border-radius:0 3px 3px 0;tab-size:2}
  h1{font-family:var(--display);font-weight:700;font-size:32px;letter-spacing:-.02em;margin:0 0 8px}
  h2{font-family:var(--display);font-weight:600;font-size:22px;margin:44px 0 10px;padding-top:22px;border-top:1px solid var(--rule)}
  h3{font-family:var(--display);font-weight:600;font-size:15px;margin:32px 0 10px;color:var(--ink-2)}
  p{margin:0 0 12px;max-width:76ch}
  header{padding:44px 0 8px}
  header p{color:var(--ink-2);font-size:17px}
  .rule{font-family:var(--display);font-size:15px;background:var(--surface);border-left:2px solid var(--accent);padding:12px 16px;margin:14px 0;max-width:76ch}
  .rule b{color:var(--ink)}
  figure{margin:14px 0}
  .figure{background:var(--surface);border:1px solid var(--rule);border-radius:6px;padding:10px 8px;max-width:var(--figw)}
  .figure svg{display:block;width:100%;height:auto}
  figcaption{font-size:13.5px;color:var(--muted);margin-top:6px;max-width:76ch}
  .fl{font-family:var(--mono);font-size:9.5px;letter-spacing:.05em;text-transform:uppercase;margin-top:5px}
  .frames{display:grid;gap:10px}
  figure.frame{margin:0}
  .item{border-top:1px solid var(--rule);padding:16px 0 4px}
  .note{font-size:15px;color:var(--ink-2);margin:0 0 8px}
  code{font-family:var(--mono);font-size:.86em;background:var(--surface-2);padding:1px 5px;border-radius:3px}
  table{border-collapse:collapse;margin:12px 0;font-family:var(--display);font-size:14px}
  td,th{text-align:left;padding:6px 20px 6px 0;border-bottom:1px solid var(--rule)}
  th{font-family:var(--mono);font-size:10px;letter-spacing:.1em;text-transform:uppercase;color:var(--muted);font-weight:500}
  /* Margin notes: absolutely positioned, no part in layout, gone when there
     is no margin to write in. */
  .annofig{position:relative;margin:16px 0;max-width:960px}
  .leads{position:absolute;left:0;top:0;pointer-events:none;overflow:visible}
  .anno{position:absolute;width:200px;pointer-events:none;
    font-family:'Caveat',cursive;font-size:18px;line-height:1.12;color:var(--pen)}
  .anno.left{right:calc(100% + 26px);text-align:right}
  .anno.right{left:calc(100% + 26px);text-align:left}
  @media (max-width:1420px){.anno,.leads{display:none}}
  .rowhit{cursor:crosshair}
  #pop{position:fixed;z-index:50;pointer-events:none;opacity:0;transition:opacity .09s;
    background:var(--surface-2);border:1px solid var(--rule-2);border-radius:7px;
    padding:10px 13px 11px;box-shadow:0 8px 28px rgba(0,0,0,.55)}
  #pop.on{opacity:1}
  #pop h4{font-family:var(--mono);font-size:9.5px;letter-spacing:.1em;text-transform:uppercase;
    color:var(--muted);margin:0 0 7px;font-weight:500}
  #pop .rows{display:grid;grid-template-columns:16px max-content max-content;
    column-gap:11px;row-gap:6px;align-items:center;font-family:var(--display);font-size:12.5px}
  #pop .p{display:contents}
  #pop .p i{display:block;justify-self:center}
  #pop .p b{font-weight:600;color:var(--ink);white-space:nowrap}
  #pop .p span{color:var(--muted);font-size:11.5px;white-space:nowrap}
  /* A section that is reference rather than argument, folded away until asked for. */
  .keylist{list-style:none;padding:0;margin:10px 0 14px;display:grid;gap:7px}
  .keylist li{display:grid;grid-template-columns:34px auto;align-items:center;gap:12px}
  .keylist span{font-family:var(--display);font-size:14.5px;color:var(--ink-2)}
  .chip{width:30px;height:8px;display:block}
  .chip.mk{width:13px;height:13px;margin:0 auto}
  .tabs{display:flex;gap:4px;padding:22px 0 0;position:sticky;top:0;z-index:20;
    background:var(--ground)}
  .tab{font-family:var(--display);font-size:13.5px;color:var(--muted);background:none;
    border:1px solid transparent;border-radius:5px;padding:6px 14px;cursor:pointer}
  .tab:hover{color:var(--ink)}
  .tab.on{color:var(--ink);background:var(--surface);border-color:var(--rule)}
  main>section>h2:first-child{border-top:0;padding-top:0;margin-top:34px}

  /* Scrubbing: one tick per state the trace was actually in. The frame is
     swapped in place and nothing around it moves. */
  .scrubbar{position:relative;height:22px;margin:12px 4px 0}
  .scrubbar .rail{position:absolute;inset:0;pointer-events:none;z-index:2}
  .scrubbar .rail::before{content:"";position:absolute;left:0;right:0;top:10px;height:2px;
    background:var(--rule-2);border-radius:2px}
  .scrubbar .tick{position:absolute;top:4px;width:2px;height:14px;margin-left:-1px;padding:0;
    border:0;border-radius:1px;background:var(--rule-2);cursor:pointer;pointer-events:auto}
  .scrubbar .tick:hover{background:var(--muted)}
  .scrubbar .tick.on{background:var(--accent);width:3px;margin-left:-1.5px}
  .scrubbar .head{position:absolute;top:2px;width:2px;height:18px;margin-left:-1px;
    background:var(--accent);opacity:.35;pointer-events:none}
  /* A stretch where nothing happened, pulled in: marked so the compression is
     visible rather than silently distorting the axis. */
  .scrubbar .brk{position:absolute;top:9px;height:4px;pointer-events:none;border-radius:2px;
    background:repeating-linear-gradient(45deg,var(--rule-2) 0 2px,transparent 2px 4px)}
  .scrubbar input{position:absolute;inset:0;z-index:1;width:100%;height:22px;margin:0;
    -webkit-appearance:none;appearance:none;background:none;cursor:grab}
  .scrubbar input:active{cursor:grabbing}
  .scrubbar input::-webkit-slider-runnable-track{height:22px;background:none}
  .scrubbar input::-moz-range-track{height:22px;background:none}
  .scrubbar input::-webkit-slider-thumb{-webkit-appearance:none;width:13px;height:13px;margin-top:4.5px;
    border-radius:50%;background:var(--accent);border:2px solid var(--ground);box-shadow:0 0 0 1px var(--accent)}
  .scrubbar input::-moz-range-thumb{width:13px;height:13px;border-radius:50%;background:var(--accent);
    border:2px solid var(--ground);box-shadow:0 0 0 1px var(--accent)}
  .scrubbar input:focus-visible{outline:2px solid var(--accent);outline-offset:2px;border-radius:4px}
  .scrub .say{display:grid;grid-template-columns:52px 1fr;gap:12px;align-items:baseline;margin:8px 4px 0}
  .scrub .at{font-family:var(--mono);font-size:10px;color:var(--muted);text-align:right;
    font-variant-numeric:tabular-nums}
  .scrub .evt{font-family:var(--display);font-size:13.5px;color:var(--ink-2)}
  .fold{margin:0}
  .fold>summary{list-style:none;cursor:pointer;display:flex;align-items:baseline;gap:12px}
  .fold>summary::-webkit-details-marker{display:none}
  .fold>summary h2{display:inline;border-top:0;padding-top:0;margin:44px 0 10px}
  .fold>summary::before{content:"▸";color:var(--muted);font-size:13px;align-self:center;margin-top:34px}
  .fold[open]>summary::before{content:"▾"}
  .fold>summary span{color:var(--muted);font-size:13.5px;margin-top:34px}
  .fold>summary:hover h2,.fold>summary:hover span{color:var(--ink)}
  .fold::before{content:"";display:block;border-top:1px solid var(--rule);margin-top:44px}
  @media (prefers-reduced-motion:reduce){*{animation:none!important;transition:none!important}}
</style>
${sidebar}
<main>
<nav class="tabs" role="tablist"><button class="tab on" data-tab="concepts">Concepts</button><button class="tab" data-tab="scenarios">Scenarios</button><button class="tab" data-tab="fixtures">Fixtures</button><button class="tab" data-tab="docs">Docs</button></nav>
<header>
  <h1>Trace Design System</h1>
  <p>How Inngest draws a function run. The vocabulary is in the panel on the left and is live: change it there and every figure on this page changes.</p>
</header>

<section id="t-concepts">
<h2>Moments</h2>
<p>Inngest records a run as spans. A span carries timestamps: when the step was enqueued, when the SDK started executing it, when it resolved. Those timestamps are what the trace has.</p>
${fig(J.primer.events,'Three timestamps from one step.')}

<h2>Durations</h2>
<p>The interval between two timestamps is what you want to read.</p>
${fig(J.primer.durations,'queuedAt to startedAt, then startedAt to endedAt.')}

<h2>A step’s lifecycle</h2>
<p>Every step reaches the same states whatever its type: enqueued, executing, resolved. A step that throws adds an attempt and a backoff and goes round again.</p>
${fig(J.primer.lifecycle,'One step.run() that threw, backed off, and returned on the second attempt.')}

<h2>Two rules</h2>
<p>Marks report whether something has resolved.</p>
${keys([[mark('hollow'),'hollow: not resolved'],[mark('ok'),'filled: resolved. Only a resolution is green or red.']])}
${fig(J.primer.resolved)}
<p>Bars report whether the SDK was executing.</p>
${keys([[chip('good'),'solid: your function was running'],[chip('waitok'),'hatched: it was not']])}
${fig(J.primer.substance)}
<div class="rule">Colour says what kind. Fill says whether the SDK was executing, for a bar, and whether it has resolved, for a mark. The full set is in the panel on the left.</div>

<h2>Structure</h2>
<p>A discovery request is the SDK reporting what to run next. When one reports more than one step, it gets its own bar and a ribbon down to the steps it reported.</p>
${fig(J.annotated.structure.svg,'Blue is the discovery request. The ribbon threads the steps it reported. Cables appear on hover or select and point at what caused the row.')}
<div class="rule">A discovery bar appears only where the request reported more than one step. A request that reported one step, which is the next step in a chain or the step after a coalesce, is a single execution and rolls into that step’s row.</div>

<h2>The Run row</h2>
<p>Today the Run row is a full width bar in the run’s status colour. The header, the status badge and the axis already report that. It could instead report where the elapsed time went.</p>
${fig(J.runbar.idea,'Grey is elapsed time with no SDK execution. Colour is time the SDK was executing, discovery requests and finalization included. Any failure in a slice colours it red — a cluster of failures among successes is the thing an overview exists to show.')}

<h2>Compressed time</h2>
<p>A run can be nine days elapsed and five seconds executing. Dead time is worth almost none of the width, and the threshold is low: anything with nothing executing for more than a few percent of the run collapses to a band, and an hour and a week get the same few pixels.</p>
${fig(EX.t1.frames[0].svg,'The cut takes the middle of the sleep, so the bar visibly begins, is torn, and resumes. The Run row’s own track tears rather than carrying a glyph laid on top of it, and what runs through the band is blurred and desaturated — it is left sharp on the Run row, because the tear is the thing saying “compressed here”.')}
<div class="rule">Only the drawing compresses. Every reported duration is still wall clock.</div>
<p>The width left over is split between the surviving stretches in proportion to their real duration, which is the same thing as saying <strong>a second is worth the same number of pixels wherever it lands</strong>. Two spans either side of a compressed gap stay comparable, and several compressions need no special case.</p>
${fig(EX.t1b.frames[0].svg,'Thirty seconds of work on one side and ten on the other: the remaining width splits 75/25.')}
<p>The bands share one budget, so a run full of idle stretches does not spend its width on the parts where nothing happened. They thin instead, and below a width that can hold them a band drops its label, then its tear, leaving a marked line. The rules and the blur never go — they are what says <em>not to scale</em>.</p>
${fig(EX.t1c.frames[0].svg,'Eight compressions sharing one budget. The polls between them still share the one scale.')}

<h2>Your server, in detail</h2>
<p>A step instrumented with <code>@inngest/otel</code> reports what your code did inside it — HTTP calls, database queries, third-party APIs. Everything above this point is Inngest&rsquo;s own timing; this is the first thing in the trace that is not.</p>
<p>They are <em>your code at finer grain</em>, so they keep the status colours a step has. What separates them from a step is <strong>nesting and weight, not a new colour</strong>: a span is a subdivision of the bar above it, never a peer of it.</p>
${frames('o0')}
<p>The same at any complexity. Spans inside a <code>Promise.all</code> overlap, and on their own pitch that reads as exactly what it is &mdash; several things running at once inside one step &mdash; without a ribbon, a mark or an ordering claim, none of which apply to something the trace did not schedule.</p>
${frames('o0b')}
<div class="rule">A span is a plain bar on a tighter pitch: no queue mark, no start, no resolution circle. The event vocabulary belongs to the trace, and a span is something that happened inside one row of it.</div>

${fig(EX.o1.frames[0].svg,'Expanded: indented, thinner, same status colours. The step is still the thing that retried — a span inside it is not separately retryable.')}
<p>The nesting is a claim, and the trace has to be able to keep it.</p>
${fig(EX.o2.frames[0].svg,'A span sits inside its step, because that is where it ran. One that outruns it is a clock disagreement between your process and ours, not a slow query, and the row says so rather than clamping quietly.')}

<h2>The selected row</h2>
<p>A row is plotted against the whole run, so a step that took ten seconds inside a twenty minute run is a few pixels wide and its marks overlap. Selecting it replots that row across the full width using the same bars and marks, and names each piece: intervals with their durations, marks with what happened. Hover a piece for its description.</p>
<div class="frames two">${J.detail.frames.map(f=>`<figure class="frame"><div class="figure">${f.svg}</div><figcaption class="fl">${f.l}</figcaption></figure>`).join('')}</div>
<div class="rule">This is not a second way of drawing a run. Same bars, same marks, more width.</div>

<h2>Annotations</h2>
<p>Optional, usually a duration. The annotation sits after the row’s last bar rather than in a fixed column, so it takes no horizontal space and disappears when there is nothing to report.</p>
${fig(J.barvocab.notes,'A duration after the bar it belongs to. The same slot can carry an attempt count, a flow control reason, or the number of runs an event started.')}

<details class="fold">
<summary><h2>Step types</h2><span>every step type and the states it reaches</span></summary>
<div class="grid">${Object.entries(J.steptypes).map(([t,v])=>`<div class="item"><p class="note"><code>${t}</code></p>${fig(v.svg,v.cap)}</div>`).join('')}</div>
</details>

</section>

<section id="t-scenarios" hidden>
<h2>Scenarios</h2>
<p>Ordered from one step to the cases that are hard to draw. Each shows the run at rest and, where it differs, on hover or select.</p>
${scenarios}

</section>

<section id="t-docs" hidden>
<h2>User docs</h2>
<p class="note">The page we would ship, rendered from <code>docs/understanding-traces.md</code> with its figures resolved to the live drawings. If a rule changes and this stops being true, that is the signal.</p>
<div class="doc">${DOC}</div>
</section>

<section id="t-fixtures" hidden>
<h2>Scrub a real run</h2>
<p>Each fixture is a captured run at its measured proportions, shown complete. Drag the slider to move back through every intermediate state: rows appear as their spans are created, intervals that have not finished are blue because their outcome is not decided yet, and the axis rescales so the elapsed part always fills the width.</p>
<div class="grid">${scrubs}</div>

<h2>Captured fixtures</h2>
<p>Runs from the captured fixture set, drawn at their measured proportions.</p>
<div class="grid">${fixtures}</div>

<h2>Connect</h2>
<p>Connect changes the transport, not the spans. Same span names, same shape, same step ops, so nothing above needs a Connect variant.</p>
${fig(J.connect.same,'The same run over HTTP and over Connect.')}
<p>Two differences sit inside the spans. A Connect run carries no <code>inngest.http.timing</code>, so there is no network breakdown to expand. The connect proxy polls for the SDK response every 2s plus a random 0 to 3s; when its push is missed that wait lands inside the execution span with nothing marking it, so the execution figure over reports.</p>
${fig(J.connect.poll)}
</section>
</main>
<div id="pop" role="tooltip"></div>
<script>
(function(){
  // Derived from the vocabulary at build time: the popover cannot describe a
  // bar with a colour or a substance the figure did not draw it with, and it
  // reads the same variables, so it follows the configurator too.
  var BAR=${JSON.stringify(V.FILL)};
  var HATCH=${JSON.stringify(Object.fromEntries(Object.keys(V.FILL).filter(k=>V.NOCOMPUTE.has(k)).map(k=>[k,1])))};
  var EVC=${JSON.stringify(V.EVC)};
  var HOLLOW=${JSON.stringify([...V.EV_HOLLOW])};
  var pop=document.getElementById('pop');

  function swatch(p){
    if(p.t==='b'){
      var v=BAR[p.k]||'var(--muted)';
      return HATCH[p.k]
        ? '<i style="width:16px;height:7px;border-radius:1px;background:repeating-linear-gradient(45deg,'+v+' 0 2px,transparent 2px 4px);opacity:.9"></i>'
        : '<i style="width:16px;height:7px;border-radius:1px;background:'+v+'"></i>';
    }
    var c=EVC[p.k]||'var(--muted)';
    if(p.k==='cancelled') return '<i style="width:9px;height:9px;margin:0 auto;background:'+c+'"></i>';
    if(HOLLOW.indexOf(p.k)>=0) return '<i style="width:11px;height:11px;margin:0 auto;border-radius:50%;border:2px solid '+c+';background:var(--surface)"></i>';
    return '<i style="width:11px;height:11px;margin:0 auto;border-radius:50%;background:'+c+'"></i>';
  }

  function show(el,ev){
    var raw=el.getAttribute('data-parts'); if(!raw) return;
    var parts; try{ parts=JSON.parse(raw.replace(/&apos;/g,"'")); }catch(e){ return; }
    pop.innerHTML='<h4>'+(el.getAttribute('data-row')||'row').trim()+'</h4>'+
      '<div class="rows">'+parts.map(function(p){ return '<div class="p">'+swatch(p)+'<b>'+p.n+'</b><span>'+p.d+'</span></div>'; }).join('')+'</div>';
    pop.classList.add('on');
    var r=pop.getBoundingClientRect(), x=ev.clientX+16, y=ev.clientY+16;
    if(x+r.width>innerWidth-12) x=ev.clientX-r.width-16;
    if(y+r.height>innerHeight-12) y=Math.max(12,ev.clientY-r.height-16);
    pop.style.left=x+'px'; pop.style.top=y+'px';
  }

  document.addEventListener('mouseover',function(e){
    var t=e.target.closest&&e.target.closest('.rowhit');
    if(t) show(t,e); else pop.classList.remove('on');
  });
  // ---- tabs -------------------------------------------------------------
  var tabs=document.querySelectorAll('.tab');
  tabs.forEach(function(t){
    t.addEventListener('click',function(){
      tabs.forEach(function(x){ x.classList.toggle('on',x===t); });
      // Derived from the tab buttons, not a hardcoded list: adding the Docs
      // tab left this at three ids, so clicking it hid every section and
      // showed none of them.
      tabs.forEach(function(x){
        var el=document.getElementById('t-'+x.dataset.tab);
        if(el) el.hidden = (x!==t);
      });
      window.scrollTo(0,0);
    });
  });

  // ---- fixture scrubbing -------------------------------------------------
  // The state at a moment is computed here rather than picked from a set of
  // pre-drawn frames, so every pixel of the scrubber has its own exact frame.
  var FX=${JSON.stringify(Object.fromEntries(J.fixtures.map(f=>[f.id,f])))};
  // Geometry and the two drawing primitives come from the vocabulary, shipped
  // as source rather than restated here: the browser draws a bar and a mark
  // with the same code the figures above were built with.
  var BINFO=${JSON.stringify(V.BAR_INFO)}, EINFO=${JSON.stringify(V.EVENT_INFO)};
  var GEOM=${JSON.stringify(V.GEOM)};
  var EV_HOLLOW=${JSON.stringify(V.EV_HOLLOW)};
  var cyOf=${V.cyOf.toString()};
  var pxOf=${V.pxOf.toString()};
  var barSvg=${V.barSvg.toString()};
  var markSvg=${V.markSvg.toString()};
  var IN_PROGRESS={good:'running', bad:'running', waitout:'wait', waitok:'wait'};
  var MONO="font-family='JetBrains Mono, ui-monospace, monospace'";

  /** Where the handle is, in real elapsed time. Harmonic inside a stretch so
      the picture advances evenly rather than time doing. */
  function timeAt(axis,p){
    if(p<=0) return axis[0][0];
    for(var i=0;i<axis.length;i++){
      var s=axis[i];
      if(p<=s[3]){
        var a=Math.max(s[0],0.15), b=Math.max(s[1],0.15);
        var u=(s[3]===s[2])?1:(p-s[2])/(s[3]-s[2]);
        return 1/((1/a)+u*((1/b)-(1/a)));
      }
    }
    return axis[axis.length-1][1];
  }

  function draw(fx,t){
    var k=100/Math.max(t,0.15), h=GEOM.TOP*2+GEOM.ROW*fx.rowCount, out='', i=0;
    var label=function(n,y){ return '<text x="2" y="'+(y+2.5)+'" '+MONO+' font-size="7" fill="var(--muted)">'+n+'</text>'; };
    fx.rows.forEach(function(r){
      var y=cyOf(i);
      if(r.run){
        var to=Math.min(t,r.end);
        out+=label('Run',y);
        out+='<rect x="'+GEOM.LBL+'" y="'+(y-GEOM.TRACK_H/2)+'" width="'+((to/100)*GEOM.PLOT*k).toFixed(2)+'" height="'+GEOM.TRACK_H+'" rx="1.2" fill="var(--rule-2)"/>';
        r.iv.forEach(function(v){
          var b=Math.min(v[1],to); if(b<=v[0]) return;
          var running=v[1]>to;
          var col=running?'var(--disc)':v[2]===0?'var(--warn)':v[2]===2?'var(--muted)':'var(--good)';
          var w=Math.max(v[2]===0&&!running?GEOM.MIN_FAIL_W:GEOM.MIN_W, ((b-v[0])/100)*GEOM.PLOT*k);
          out+='<rect x="'+pxOf(v[0],k).toFixed(2)+'" y="'+(y-GEOM.RUN_H/2)+'" width="'+w.toFixed(2)+'" height="'+GEOM.RUN_H+'" rx="1.2" fill="'+col+'"/>';
        });
        out+=markSvg(GEOM.LBL,y,'queued');
        if(r.ra!=null && t>=r.ra) out+=markSvg(GEOM.LBL+(to/100)*GEOM.PLOT*k,y,r.rs);
        i++; return;
      }
      var segs=[], parts=[];
      r.segs.forEach(function(g){
        if(g[1]>=t) return;
        var w=Math.min(g[2],t-g[1]);
        segs.push([w<g[2]?(IN_PROGRESS[g[0]]||g[0]):g[0], g[1], w]);
      });
      if(!segs.length) return;
      var marks=r.marks.filter(function(m){ return m[1]<=t+0.001; });
      out+=label(r.n,y);
      segs.forEach(function(g){ out+=barSvg(g[0],g[1],g[2],y,{k:k}); });
      marks.forEach(function(m){ out+=markSvg(+pxOf(m[1],k).toFixed(2),y,m[0]); });
      // the same decomposition the row popover shows, built from what is drawn
      var mi=0;
      segs.forEach(function(g){
        while(mi<marks.length && marks[mi][1]<=g[1]+0.001){
          var e=EINFO[marks[mi][0]]; if(e) parts.push({t:'e',k:marks[mi][0],n:e[0],d:e[1]});
          mi++;
        }
        var kk=g[0].replace(/[!*]+$/,''), bb=BINFO[kk];
        if(bb) parts.push({t:'b',k:kk,n:bb[0],d:bb[1]});
      });
      for(;mi<marks.length;mi++){ var e2=EINFO[marks[mi][0]]; if(e2) parts.push({t:'e',k:marks[mi][0],n:e2[0],d:e2[1]}); }
      out+='<rect class="rowhit" x="'+(GEOM.LBL-6)+'" y="'+(y-8)+'" width="'+(GEOM.PLOT+12)+'" height="16" fill="transparent" data-row="'+r.n+'" data-parts="'+JSON.stringify(parts).replace(/"/g,'&quot;')+'"/>';
      i++;
    });
    // before anything is known: the run, and a step enqueued but not reported
    if(i===1) out+=label('?',cyOf(1))+barSvg('idle',0,t,cyOf(1),{k:k})+markSvg(GEOM.LBL,cyOf(1),'queued');
    return '<svg viewBox="0 0 '+GEOM.W+' '+h+'" role="img">'+out+'</svg>';
  }

  document.querySelectorAll('.scrub').forEach(function(s){
    var fx=FX[s.dataset.fx], slot=s.querySelector('.frame'),
        input=s.querySelector('input'), at=s.querySelector('.at'), ev=s.querySelector('.evt'),
        ticks=[].slice.call(s.querySelectorAll('.tick')), head=s.querySelector('.head');
    function show(pos){
      var t=timeAt(fx.axis,pos);
      slot.innerHTML=draw(fx,t);
      at.textContent=t.toFixed(1)+'%';
      var e=fx.events[0];
      for(var j=0;j<fx.events.length;j++) if(fx.events[j].t<=t+0.001) e=fx.events[j];
      ev.textContent = pos>=100 ? e.text+' \u00b7 complete' : e.text;
      ticks.forEach(function(x){ x.classList.toggle('on', Math.abs(+x.dataset.p-pos)<0.4); });
      head.style.left=pos+'%';
      if(+input.value!==Math.round(pos*100)) input.value=Math.round(pos*100);
    }
    input.addEventListener('input',function(){ show(+input.value/100); });
    ticks.forEach(function(x){ x.addEventListener('click',function(){ show(+x.dataset.p); }); });
    show(100);
  });

  // ---- configurator ----------------------------------------------------
  // Nothing is redrawn: every choice is a variable on :root, and the figures
  // already read their colours and substances from those variables.
  var PAL=${JSON.stringify(PALETTES)}, STY=${JSON.stringify(STYLES)}, WT=${JSON.stringify(WEIGHTS)};
  var root=document.documentElement, KEY='tds.vocab.v1';
  var state={};
  try{ state=JSON.parse(localStorage.getItem(KEY)||'{}'); }catch(e){ state={}; }

  function apply(){
    root.removeAttribute('style');
    for(var k in state) if(k.charAt(0)==='-') root.style.setProperty(k,state[k]);
    document.querySelectorAll('#side .sw,#side .pill').forEach(function(b){
      var on=false;
      if(b.dataset.set)    on = state[b.dataset.set]==='var(--'+b.dataset.tag+')';
      if(b.dataset.style)  on = (state['--sub-'+b.dataset.style+'-bg']||'') === String(STY[b.dataset.val][0]);
      if(b.dataset.weight) on = (state['--ev-'+b.dataset.weight+'-w']||'') === String(WT[b.dataset.val]);
      if(b.dataset.pal)    on = (state.__pal||'default')===b.dataset.pal;
      b.classList.toggle('on',!!on);
    });
    try{ localStorage.setItem(KEY,JSON.stringify(state)); }catch(e){}
  }
  function set(k,v){ if(v==null) delete state[k]; else state[k]=v; }

  // Sliders write the same state the swatches do, so reset clears them too.
  var SL=${JSON.stringify(SLIDERS)};
  document.querySelectorAll('#side input[data-geo]').forEach(function(inp){
    inp.addEventListener('input',function(){
      var k=inp.dataset.geo;
      set(k, inp.value+'px');
      var out=document.querySelector('[data-out="'+k+'"]');
      if(out) out.textContent=inp.value;
      apply();
    });
  });
  function syncSliders(){
    SL.forEach(function(sl){
      var inp=document.querySelector('input[data-geo="'+sl.k+'"]');
      if(!inp) return;
      var v=state[sl.k]!=null?parseFloat(state[sl.k]):sl.d;
      inp.value=v;
      var out=document.querySelector('[data-out="'+sl.k+'"]');
      if(out) out.textContent=v;
    });
  }

  document.getElementById('side').addEventListener('click',function(e){
    var b=e.target.closest('button'), row=e.target.closest('.vhead');
    if(!b){ if(row) row.parentNode.classList.toggle('open'); return; }
    var d=b.dataset;
    if(b.id==='reset'){ state={}; apply(); return; }
    if(d.pal){
      var p=PAL[d.pal]; state.__pal=d.pal;
      for(var t in p) set('--'+t,p[t]);
    }
    // A second click on the active choice clears it, back to the default.
    else if(d.set)    set(d.set, b.classList.contains('on') ? null : d.val);
    else if(d.style){ var s=STY[d.val];
      var off=b.classList.contains('on');
      set('--sub-'+d.style+'-bg', off?null:String(s[0]));
      set('--sub-'+d.style+'-line',off?null:String(s[1])); }
    else if(d.weight) set('--ev-'+d.weight+'-w', b.classList.contains('on')?null:String(WT[d.val]));
    apply();
  });
  apply();

  document.addEventListener('mousemove',function(e){
    if(!pop.classList.contains('on')) return;
    var r=pop.getBoundingClientRect(), x=e.clientX+16, y=e.clientY+16;
    if(x+r.width>innerWidth-12) x=e.clientX-r.width-16;
    if(y+r.height>innerHeight-12) y=Math.max(12,e.clientY-r.height-16);
    pop.style.left=x+'px'; pop.style.top=y+'px';
  });
})();
</script>`;

fs.writeFileSync(HERE+'../trace-design-system.html',page);
console.log('bytes',page.length,'| svgs',(page.match(/<svg /g)||[]).length);
