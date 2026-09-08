const ESBUILD=new URL('../../../../ui/node_modules/.pnpm/node_modules/.bin/esbuild',import.meta.url).pathname;
const UI_MODULES=new URL('../../../../ui/node_modules',import.meta.url).pathname;
const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import { execFileSync } from 'child_process';
import * as V from './vocabulary.mjs';
import * as R from './rules.mjs';
// The panel opens where the rule sits, rather than restating the number.
const MULT_DEFAULT=R.ELASTIC.computeMultiple;
// Cleared before the generators run; each appends what the elastic rule could
// not reach, so an empty file means every figure is derived rather than placed.
try{ fs.unlinkSync(new URL('./unruled.json',import.meta.url).pathname); }catch{}
const GENS=['vocab','barvocab','items-a','items-bc','items-disc','items-more','connect','batch1','runbar','annotated','steptypes','detail','primer'];
/**
 * Four builds, because two features change what is drawn.
 *
 * `hide the opening queue` re-lays the whole trace and `compress dead time`
 * moves every bar, so neither can be a CSS variable the way the spacing
 * sliders are. Each combination is generated and the page carries whichever
 * figures actually differ between them — most do not, and those stay single.
 */
/**
 * One build, and the events every figure was drawn from.
 *
 * The page used to carry four builds of each figure -- one per combination of
 * the features that change what is drawn -- and swap between them. They are
 * properties of the component now, so it re-renders instead, and there is one
 * of everything again.
 */
// The frameless build's figures are exported and then inlined into the Docs
// tab by the framed one, so the two must not number their ids alike: the same
// id defining two different clip paths resolves to whichever came first.
const UIDBASE=process.env.DS_FRAME==='0'?500000:0;
GENS.forEach((g,gi)=>execFileSync('node',[HERE+g+'.mjs'],
  {stdio:'pipe', env:{...process.env, DS_UID_BASE:String(UIDBASE+gi*1000)}}));
const J=Object.fromEntries(GENS.map(g=>[g,JSON.parse(fs.readFileSync(HERE+g+'.json','utf8'))]));

/** id -> the events that figure was drawn from, gathered from every generator. */
const FIGDATA=(()=>{
  const out={};
  for(const f of fs.readdirSync(HERE)) if(/^figdata-\d+\.json$/.test(f))
    for(const d of JSON.parse(fs.readFileSync(HERE+f,'utf8'))) out[d.id]=d;
  return out;
})();

/** The renderer and its payload translator, bundled for the page. */
const kit=(()=>{
  const out=HERE+'.kit.js';
  execFileSync(ESBUILD,[HERE+'trace/entry.jsx','--bundle','--format=iife','--jsx=automatic',
    '--external:fs','--minify','--define:process.env.NODE_ENV="production"',
    '--outfile='+out,'--log-level=error'],
    {cwd:HERE, env:{...process.env, NODE_PATH:UI_MODULES}, stdio:'pipe'});
  const js=fs.readFileSync(out,'utf8');
  fs.unlinkSync(out);
  return js;
})();

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
  // A lone frame has no other state to be distinguished from, so its caption is
  // dropped: "at rest" under every figure on the page is nine words of nothing.
  const solo=e.frames.length===1;
  return `<div class="frames${e.frames.length>1?' two':''}">`+
    e.frames.map(f=>`<figure class="frame"><div class="figure">${f.svg}</div>`+
      (solo?'':`<figcaption class="fl">${f.l}</figcaption>`)+`</figure>`).join('')+`</div>`;};

/**
 * The Scenarios page: one code example, then everything worth saying about it.
 *
 * Ordered from a single step to the cases that are hard to draw, and grouped by
 * what you WROTE. Grouping by concept meant reprinting the same code under
 * every heading it touched and left the same fact scattered across a dozen
 * figures — attribution said four times, discovery requests said five.
 */
const SCEN=[
 ['oneStep',[
  ['c14',['Queue time: the step enqueued, waiting for the executor to pick it up. Hatched, because it is not your code running.',
          'Below it, Finalization: the request that asks what runs next and is told nothing. It is your compute, so it keeps its own row — and it is a row like any other, drawn only where the run actually got that far. A run still going, cancelled, or ended by a failure has none.']],
  ['c16','Held by flow control — concurrency, throttle, rate limit or debounce. Queue time with a reason, and a colour of its own.'],
  ['c25',['It threw, backed off, and returned on the second attempt. The backoff is red-hatched: a consequence of the failure, not a failure itself.',
          'Every millisecond between the row’s start and its resolution belongs to a named interval — the queue, the attempt, its backoff, the queue again, and the attempt that worked. There are no unexplained gaps.']],
  ['c23','It threw on the final attempt, and the run failed with it.'],
  ['c30','Every attempt threw. The only place a circle is filled red.'],
  ['h2','A row’s label and its drawing have to agree about which interval the number names.'],
 ]],
 ['chain',[
  ['c0','One request, one step. The SDK reports and executes in the same request, so there is no discovery bar — but the execution still BEGAN not knowing what it would run, and the start mark says so.'],
  ['c15','Where the request was a separate execution it gets a bar at the head of the row it reported.'],
  ['c21','The gap between one step resolving and the next request starting is drawn once — as the cable, and as the interval.'],
 ]],
 ['caught',[
  ['c24','The function caught the error and carried on. Red step, green run: the outcomes are separate facts.'],
 ]],
 ['parallel',[
  ['c1',['One request reports three steps. The ribbon threads their enqueue marks, which are blue because a discovery request put them there.',
         'It covers exactly the steps that request reported, so an uneven fan-out is countable without interacting. Nothing static predicted that width.']],
  ['c2','All three resolving causes the next request. That request reported one step, so it rolls into that step’s row.'],
  ['c16b','Flow control on three steps at once is the same fact three times, and takes the same treatment.'],
  ['i1','Three tiers of attention: the row, what caused it, everything else. Nothing is removed, only quietened.'],
  ['h1','Where nothing reported what caused a request, the trace declines rather than guessing — some earlier step always finished just before.'],
 ]],
 ['discovery',[
  ['c6','One resumption reports two steps. Both sit under one ribbon, so no order is implied between them.'],
  ['c67','The request that would report the next step threw twice before it succeeded. It stays discovery blue: this was Inngest’s failure, not yours.'],
  ['c68','It never succeeded, so no step was ever created. This is the one case where a request gets a row of its own.'],
 ]],
 ['nested',[
  ['c3','Fan-out inside a fan-out. One ribbon per discovery request.'],
  ['c8','Two branches that each caused their own request. Two requests, and no ribbon: nothing here was reported together.'],
 ]],
 ['orphan',[
  ['c10','A step started but never awaited. Nothing downstream depended on it, so it has no outgoing cable — the absence is the drawing.'],
 ]],
 ['emit',[
  ['c31','step.sendEvent() is the one row that points out of the run. The ring is hollow because those runs are not in this trace, and the count is the way into them.'],
 ]],
 ['sleep',[
  ['t1','Seven days of dead time, given four percent of the width. The threshold is scale-free: a stretch is compressed when it is longer than all the compute in the run put together, several times over. Switch it off in the panel to see what it is saving you.'],
  ['t1d','Asleep right now. The bar is open and carries no closing mark, and there is no Finalization row: the run has not ended.'],
  ['t4','A step too short to draw is still drawn, at a minimum width, with its real duration beside it.'],
 ]],
 ['wait',[
  ['w1','A wait that matched. Blue while it is open, green at the mark where the event arrived.'],
  ['w2','A wait that expired with no match. The run carries on: a timeout is a result the function can act on.'],
  ['c28','A wait still open when the run was cancelled. Undecided, so it ends on the square rather than a resolution.'],
 ]],
 ['otel',[
  ['o0','A step instrumented with @inngest/otel. The icon by the name says there are spans inside; selecting the row expands them.'],
  ['o1','Expanded. Everything above this point is Inngest’s own timing; these rows are the first thing in the trace that is not.'],
  ['o0b','Parallel work inside one step. Collapsed it looks like any other step; expanded, the spans overlap — which is why they cannot be a rail of notches on the bar.'],
  ['o2','A span that reported no status. OpenTelemetry’s Unset is its own outcome and is not guessed into success.'],
 ]],
];

/** Shapes we are still working out. Kept apart so the page above stays settled. */
const SCEN_EXP=[
 ['loop',[
  ['s1','Five hundred sequential steps collapse to one row that reports the count and draws where each member ran.'],
 ]],
 ['wideFanout',[
  ['s2','A wide fan-out collapses the same way. The stagger of a concurrency limit is visible without expanding it.'],
 ]],
 ['failureCluster',[
  ['s3','A failure cluster two thirds through a long run is a red smear on the Run row, findable without interacting.'],
 ]],
 ['expandGroup',[
  ['s4','About forty rows at rest whatever the step count, and any group expands where you are standing.'],
 ]],
 ['manyCompressions',[
  ['t1c','Many compressions sharing one budget. Each band thins rather than the trace spending its width on nothing.'],
 ]],
];

const group = ([k,items])=>
`  <section class="sc" data-sc="${k}">
    <div class="item">
      <div class="col-l">${codeOf(k)}</div>
      <div class="col-r">
${items.map(([id,note])=>`        <div class="bit" data-fig="${id}">`+
      // One drawing can carry several points. Repeating the figure once per
      // sentence put four identical pictures under one code example.
      (Array.isArray(note)
        ? `<ul class="note">${note.map(n=>`<li>${n}</li>`).join('')}</ul>`
        : `<p class="note">${note}</p>`)+
      frames(id)+`</div>`).join('\n')}
      </div>
    </div>
  </section>`;

const scenarios = SCEN.map(group).join('\n');
/** Shapes still being worked out, kept behind their own heading. */
const experimental = `  <section class="sc exp"><h3>Experimental</h3>
    <p class="note">Shapes we are still working out. Nothing here is settled.</p>
  </section>
`+SCEN_EXP.map(group).join('\n');
/**
 * Each fixture is one function captured twice -- with checkpointing and
 * without -- so the tabs switch between two pictures of the same run rather
 * than between two runs. The caption belongs to the shape, not to either
 * capture, so it sits above both.
 */
const fixtures=Object.entries(J.batch1).map(([id,f])=>{
  // The note and the function's own source on the left, the two captures on the
  // right. The code is extracted from the file that defines the shape, so it is
  // the function that produced the run beside it rather than a transcription
  // of it.
  return `    <div class="item fixpair" data-fix="${id}">`+
    `<div><p class="note"><code>${id}</code> ${f.cap||''}</p>`+
      (f.code?`<pre class="code">${highlight(f.code)}</pre>`:'')+`</div>`+
    `<div class="modewrap">`+
      `<div class="modes">`+f.modes.map((m,i)=>
        `<button class="modeb${i?'':' on'}" data-mode="${i}">${m.mode}</button>`).join('')+`</div>`+
      f.modes.map((m,i)=>`<div class="mode${i?'':' on'}" data-mode="${i}">${fig(m.svg)}</div>`).join('')+
    `</div></div>`;
}).join('\n');

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
  {k:'--geo-gap',  n:'gap between segments',   d:0,  lo:0, hi:8, st:0.25},
];

/** Things that can be taken away entirely, to see what the row reads like. */
const TOGGLES = [
  {k:'--show-ev', n:'events', on:'inline', off:'none'},
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

/**
 * A collapsible panel section. The heading is the control, so a long panel can
 * be folded down to the part you are working in.
 */
const sec=(title,sub,body)=>
  `<section class="sec"><h5><button class="sech" data-sec="${title}" aria-expanded="true">`+
  `<span class="caret">›</span>${title}${sub?`<em>${sub}</em>`:''}</button></h5>`+
  `<div class="secb">${body}</div></section>`;

/**
 * Things the design DOES for you, which you can switch off to see what it was
 * doing. Unlike the sliders these change what is drawn, not how it is spaced,
 * so the page carries a variant of every affected figure and swaps between them.
 */
const FEATURES = [
  {k:'trim',     n:'hide the opening queue', d:'name a run’s first wait instead of drawing it'},
  {k:'compress', n:'compress dead time',     d:'collapse a stretch with nothing executing to a band'},
  {k:'compressCompute', n:'compress compute too', d:'…and a stretch of work that dwarfs the rest of the run'},
];

/**
 * The renderer, shipped to the page.
 *
 * The page draws every figure through the same component, so the renderer has
 * to be there rather than restated.
 *
 * Data URLs rather than a bundle: the modules already are modules, and rewriting
 * their relative imports to bare specifiers is enough for the import map to
 * resolve them. Nothing is transformed on the way in — what the page runs is
 * what the build ran, byte for byte apart from the specifiers.
 */
const MODULES=['rules','vocabulary','micro'];
const dsmod=`<script type="module">
  /**
   * Every figure on the page, drawn by the component.
   *
   * The build draws each one once so the page paints without waiting for
   * script; this then takes them over, rendering the SAME events through the
   * SAME renderer. It is what makes the feature toggles properties of a
   * drawing: switching one re-renders every figure instead of swapping between
   * four pre-built copies of it.
   */
  import * as micro from 'ds:micro';
  import * as rules from 'ds:rules';
  window.DS = {fig:micro.fig, layout:micro.layout, RESOLVED:rules.RESOLVED,
               setFeatures:rules.setFeatures};

  const K = window.TraceKit;
  const DATA = window.__FIGDATA || {};
  let mounts = null;

  window.__renderFigures = function(feat){
    if(!K) return;
    // Found once. After the first render the DOM holds the component's output,
    // whose figures carry ids from the bundle's own counter -- so re-reading
    // data-fig on the second pass finds nothing and the toggle does nothing.
    if(!mounts){
      mounts=[];
      for(const el of document.querySelectorAll('svg[data-fig]')){
        const d=DATA[el.dataset.fig];
        if(!d) continue;
        const host=el.parentNode;
        // Concepts figures ignore the toggles. That page demonstrates one mark
        // or one bar at a time, and a feature switched off elsewhere would
        // quietly change what the vocabulary says a thing looks like.
        const fixed=!!el.closest('#t-concepts');
        mounts.push({host, d, fixed, root:K.createRoot(host)});
      }
    }
    for(const m of mounts)
      m.root.render(K.React.createElement(K.Trace, {
        key: m.d.id,
        rows: m.d.rows, extra: m.d.extra, under: m.d.under, label: m.d.label,
        ...m.d.opts, ...(m.fixed ? {trim:true, compress:true, compressCompute:false, multiple:${MULT_DEFAULT}} : feat),
      }));
  };

<\/script>`;
const bundle=(()=>{
  const url=name=>{
    let src=fs.readFileSync(HERE+name+'.mjs','utf8')
      .replace(/from '\.\/(\w[\w-]*)\.mjs'/g, (m,d)=>`from 'ds:${d}'`);
    return 'data:text/javascript;base64,'+Buffer.from(src,'utf8').toString('base64');
  };
  const map={}; for(const n of MODULES) map['ds:'+n]=url(n);
  return `<script type="importmap">${JSON.stringify({imports:map})}<\/script>`;
})();

const sidebar=`<aside id="side">
  <svg width="0" height="0" aria-hidden="true" style="position:absolute">${V.HATCH}</svg>
  <div class="sh">
    <button id="fold" title="Collapse the panel" aria-label="Collapse the panel">‹</button>
    <strong>Vocabulary</strong>
    <button id="reset" title="Back to the proposed defaults">reset</button>
  </div>
  ${sec('Features','what the design does for you, on and off',
    FEATURES.map(f=>`<div class="tg"><label>${f.n}<em>${f.d}</em></label>`+
      `<button class="tgb on" data-feat="${f.k}">on</button></div>`).join('')+
    /* The one rule worth arguing with from the panel: how much longer than the
       run's whole compute a stretch has to be before it is worth compressing.
       Lower compresses more eagerly; at 0 every dead stretch goes. */
    `<div class="sl"><label>dead time worth compressing<b data-out="mult">${MULT_DEFAULT}×</b></label>
    <input type="range" data-mult min="0" max="12" step="0.5" value="${MULT_DEFAULT}"></div>
    <p class="foot">A stretch with nothing executing is compressed when it is
    this many times longer than <em>all</em> the compute in the run put together.
    Scale-free on purpose: a flat fraction of the run would compress ordinary
    queue intervals, which are short and are exactly what the trace is for.</p>`)}
  ${sec('Palette','',
    `<div class="pills">${Object.keys(PALETTES).map((p,i)=>
      `<button class="pill${i?'':' on'}" data-pal="${p}">${p}</button>`).join('')}</div>`)}
  ${sec('Spacing','every dimension, down to nothing',
    SLIDERS.map(sl=>`<div class="sl"><label>${sl.n}<b data-out="${sl.k}">${sl.d}</b></label>
    <input type="range" data-geo="${sl.k}" min="${sl.lo}" max="${sl.hi}" step="${sl.st||0.5}" value="${sl.d}"></div>`).join('')+
    TOGGLES.map(t=>`<div class="tg"><label>${t.n}</label>
    <button class="tgb on" data-tg="${t.k}" data-on="${t.on}" data-off="${t.off}">on</button></div>`).join(''))}
  ${sec('Bars','colour = kind of work; fill = SDK executing',
    (()=>{const seen=new Set();
      return Object.entries(V.BAR_INFO).filter(([,i])=>{
        if(seen.has(i[0])) return false; seen.add(i[0]); return true; }).map(barRow).join('');})())}
  ${sec('Events','colour = kind of moment; hollow = not resolved; ring = how we heard',
    Object.entries(V.EVENT_INFO).map(evRow).join('')+
    /**
     * The third channel, documented beside the two it does not touch. Fill
     * says whether the row has resolved and colour says what happened; how we
     * came to KNOW a moment is orthogonal to both and can land on any of them,
     * which is why it is a ring rather than another colour or another shape.
     */
    `<div class="vrow"><div class="vhead">
      <svg class="vsw ev" viewBox="0 0 15 15" aria-hidden="true">${V.dot(7.5,7.5,'hollow',1,3.4,false,true)}</svg>
      <b>reported by checkpoint</b><code>ring</code>
      <span>the SDK ran this step inline and told us while the request was still open,
      rather than the executor scheduling it and being told in the response</span>
    </div></div>`+
    `<p class="foot">Hollow and filled is a rule, not a preference. It reports whether
    the row has resolved, so it is not adjustable. The ring is the same kind of rule:
    it says the moment reached us by a different route, and it can land on any of
    the marks above.</p>`)}
</aside>`;



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
  /* A section folds from its heading. The caret is the only affordance the
     panel needs: the heading already reads as the thing it belongs to. */
  #side .sec{margin:0}
  #side .sec h5{margin:18px 0 0}
  #side .sech{all:unset;cursor:pointer;display:flex;flex-direction:column;gap:2px;
    width:100%;padding:4px 0;position:relative;padding-left:14px}
  #side .sech:hover{color:var(--ink-2)}
  #side .sech:focus-visible{outline:1px solid var(--accent);outline-offset:2px}
  #side .caret{position:absolute;left:0;top:4px;font-size:13px;line-height:1;
    transform:rotate(90deg);transform-origin:center;transition:transform .12s;color:var(--rule-2);filter:brightness(1.7)}
  #side .sech[aria-expanded="false"] .caret{transform:rotate(0deg)}
  #side .secb{padding-top:8px}
  #side .sech[aria-expanded="false"]+.secb,
  #side .sec:has(.sech[aria-expanded="false"]) .secb{display:none}
  /* A feature toggle carries a line of explanation, so its label stacks. */
  #side .tg label{display:flex;flex-direction:column;gap:1px}
  #side .tg label em{font-style:normal;font-size:11px;color:var(--rule-2);filter:brightness(1.7);line-height:1.35}
  #side .tg{align-items:flex-start}
  /* Only the variant matching the current feature toggles is shown. A figure
     that draws the same thing whatever they are was never wrapped. */
  /* --w is what a bar's width comes from, so it has to be a registered custom
     property or it cannot be transitioned — an unregistered one is a string as
     far as animation is concerned, and jumps. */
  @property --w{syntax:'<length>';inherits:false;initial-value:0px}
  /* --w has to be a REGISTERED custom property or it cannot be interpolated
     at all: an unregistered one is a string to the animation engine, and a
     bar would jump from one width to the other rather than travelling. */
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
  .tg{display:flex;justify-content:space-between;align-items:center;padding:5px 0 8px}
  .tg label{font-family:var(--display);font-size:11.5px;color:var(--muted)}
  .tgb{font-family:var(--mono);font-size:10.5px;padding:2px 9px;border-radius:3px;
    border:1px solid var(--rule-2);background:none;color:var(--muted);cursor:pointer}
  .tgb.on{color:var(--ink);border-color:var(--ink-2)}
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
  /* No overflow escape: refit() sizes the viewBox to the content, so nothing
     needs to spill any more — and while it could, a long annotation ran out of
     the side of its own box. */
  svg rect.bar{height:var(--geo-bar,7px);transform:translateY(calc(var(--geo-bar,7px) / -2))}
  svg rect.bar.sm{height:var(--geo-sbar,4.2px);transform:translateY(calc(var(--geo-sbar,4.2px) / -2))}
  svg rect.bar.rail{height:1.8px;transform:translateY(-0.9px)}
  /* A bar takes the gap off its own right edge, so segments separate without
     any of them moving. */
  svg rect.bar{width:max(0.5px, calc(var(--w,4px) - var(--geo-gap,0px)))}
  svg circle.ev,svg circle.ev-bg{display:var(--show-ev,inline)}
  svg circle.ev{r:var(--geo-mark,3px)}
  svg circle.ev-bg{r:calc(var(--geo-mark,3px) + 1.3px)}
  /* Each row knows how many gaps of each pitch sit above it. */
  svg g.r{transform:translateY(calc((var(--geo-row,17px) - 17px) * var(--i,0)
                                 + (var(--geo-span,9px) - 9px) * var(--s,0)))}
  /* The framed body sits a whole number of pitches below the Run row. */
  svg g.dy{transform:translateY(calc(var(--geo-row,17px) * var(--a,0)))}
  /* A compressed stretch is marked across every row, so its height is the
     figure's — which moves with the pitch sliders. */
  svg rect.cmpband,svg rect.gut{height:var(--fig-h)}
  svg g.cmpmid{transform:translateY(calc((var(--fig-h) - var(--fig-h0)) / 2))}
  /* A ribbon spans N row gaps, so it restretches with the pitch. */
  svg rect.run-track{height:var(--geo-trk,5px);transform:translateY(calc(var(--geo-trk,5px) / -2))}
  svg rect.run-slice{height:var(--geo-run,8px);transform:translateY(calc(var(--geo-run,8px) / -2))}
  svg rect.rib{height:calc(var(--geo-row,17px) * var(--n,1));
    transform:translateY(calc((var(--geo-row,17px) - 17px) * var(--i,0)))}
  /* The hit target is a row's worth of height, so it never hangs below the
     drawing and holds the box open at a tight pitch. */
  svg rect.rowhit{height:max(6px, var(--geo-row,17px));
    transform:translateY(calc(max(6px, var(--geo-row,17px)) / -2))}
  /* The selection band is the row it belongs to, so it takes the row's pitch —
     less a hair, so consecutive selected rows do not fuse into one block. */
  svg rect.selband{height:max(5px, calc(var(--geo-row,17px) - 2px));
    transform:translateY(calc(max(5px, calc(var(--geo-row,17px) - 2px)) / -2))}
  svg rect.selband.sp{height:max(4px, calc(var(--geo-span,9px) - 1px));
    transform:translateY(calc(max(4px, calc(var(--geo-span,9px) - 1px)) / -2))}
  /* A cable spans two rows, so it is scaled about its own start: the far end
     lands on the row it belongs to whatever the pitch is. */
  svg g.cable{transform:translateY(calc((var(--geo-row,17px) - 17px) * var(--a,0)))
                        scaleY(calc(var(--geo-row,17px) / 17px))}
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
  .grid .item,.sc .item{border-top:1px solid var(--rule);padding:16px 0 10px;min-width:0;
    display:grid;grid-template-columns:minmax(280px,360px) 1fr;gap:28px;align-items:start}
  /* The code stays with you while everything said about it scrolls past.
     Offset by the tab bar, which is sticky at the top of the page. */
  .col-l{position:sticky;top:64px;min-width:0}
  .col-r{min-width:0;display:grid;gap:10px}
  .col-l .note{margin:10px 0 0}
  /* One code example, then prose and figure alternating down the right. The
     gap between them is the only thing separating two points about the same
     code, so it has to be bigger than the gap inside one. */
  .sc{margin:0}
  .sc .item{padding:26px 0 18px}
  .sc .col-r{gap:26px}
  .sc .bit{min-width:0}
  .sc .bit .note{margin:0 0 9px;max-width:70ch}
  /* Several points about one drawing. The marker sits in the gutter so the text
     block keeps the same left edge as a single-sentence note. */
  ul.note{padding-left:1.1em;list-style:disc}
  ul.note li{margin:0 0 5px}
  ul.note li::marker{color:var(--muted)}
  @media (max-width:1080px){
    .grid .item,.sc .item{grid-template-columns:1fr;gap:12px}
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
  /* One fixture, two captures: only the selected one is in the layout, so the
     item does not reserve space for the other. */
  .modes{display:flex;gap:6px;margin:0 0 10px}
  .modeb{font:11px/1 var(--mono);letter-spacing:.02em;padding:5px 9px;border-radius:5px;
    border:1px solid var(--line);background:transparent;color:var(--muted);cursor:pointer}
  .modeb:hover{color:var(--ink)}
  .modeb.on{background:var(--surface-2);color:var(--ink);border-color:var(--muted)}
  .modewrap > .mode{display:none}
  .modewrap > .mode.on{display:block}
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
  /* The duration sits with the name, quieter, so the column still scans. */
  #pop .p b u{text-decoration:none;color:var(--muted);font-weight:400;margin-left:6px;
    font-family:var(--mono);font-size:11px}
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
<p>Organised by what you wrote. One code example, then everything the trace shows for it — ordered from a single step to the cases that are hard to draw. Where a figure has a second state, it is the hover or selection.</p>
${scenarios}
${experimental}

</section>

<section id="t-docs" hidden>
<h2>User docs</h2>
<p class="note">The page we would ship, rendered from <code>docs/understanding-traces.md</code> with its figures resolved to the live drawings. If a rule changes and this stops being true, that is the signal.</p>
<div class="doc">${DOC}</div>
</section>

<section id="t-fixtures" hidden>


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
<script>${kit}<\/script>
${bundle}
${dsmod}
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
      '<div class="rows">'+parts.map(function(p){
        // How long the interval was, where the bar came from a run that was
        // measured. Not the drawn width: the axis it was drawn on may be
        // compressed, so the pixels are not the time.
        var t=p.ms?'<u>'+p.ms+'</u>':'';
        return '<div class="p">'+swatch(p)+'<b>'+p.n+t+'</b><span>'+p.d+'</span></div>'; }).join('')+'</div>';
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
  // ---- fixture modes ----------------------------------------------------
  // A hidden figure cannot be measured, so the one being revealed is refit
  // after the swap rather than before it.
  document.querySelectorAll('.fixpair').forEach(function(item){
    item.querySelectorAll('.modeb').forEach(function(btn){
      btn.addEventListener('click',function(){
        var want=btn.dataset.mode;
        item.querySelectorAll('.modeb').forEach(function(b){ b.classList.toggle('on', b===btn); });
        item.querySelectorAll('.mode').forEach(function(m){ m.classList.toggle('on', m.dataset.mode===want); });
        refit();
      });
    });
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
      refit();
    });
  });

  // ---- fixture scrubbing -------------------------------------------------
  // The state at a moment is computed here rather than picked from a set of
  // pre-drawn frames, so every pixel of the scrubber has its own exact frame.
  // One set of runs. The features that change what is drawn are properties of
  // the component now, so there is nothing to switch between.
  var FIGDATA=${JSON.stringify(FIGDATA)};
  window.__FIGDATA=FIGDATA;
  // ---- configurator ----------------------------------------------------
  // Nothing is redrawn: every choice is a variable on :root, and the figures
  // already read their colours and substances from those variables.
  // Panel sections fold from their heading, and remember which are folded.
  var SEC='tds.folded.v1', folded={};
  try{ folded=JSON.parse(localStorage.getItem(SEC)||'{}'); }catch(e){ folded={}; }
  document.querySelectorAll('#side .sech').forEach(function(h){
    var name=h.dataset.sec;
    if(folded[name]) h.setAttribute('aria-expanded','false');
    h.addEventListener('click',function(){
      var open=h.getAttribute('aria-expanded')!=='false';
      h.setAttribute('aria-expanded', open?'false':'true');
      folded[name]=open;
      try{ localStorage.setItem(SEC,JSON.stringify(folded)); }catch(e){}
    });
  });

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
    // After the variables land, so the measurement sees the new geometry.
    refit(); requestAnimationFrame(refit);
  }
  function set(k,v){ if(v==null) delete state[k]; else state[k]=v; }

  /**
   * Refit every figure to what it actually contains.
   *
   * A figure's viewBox height is fixed when it is generated, so once the panel
   * can move the geometry the box is wrong in both directions — content spills
   * out of it at a loose pitch and it keeps empty space at a tight one. Guessing
   * the difference from the row count was close and never right.
   *
   * getBBox() is the actual answer: it reports the union of what is drawn, in
   * user units, with the CSS transforms applied. The width is left alone so
   * figures stay aligned with each other; only the height follows.
   */
  /**
   * What a figure is as tall as.
   *
   * A compression band is a backdrop drawn across the rows, starting above the
   * first one, and measuring it made a compressed figure about 12px taller than
   * the same figure uncompressed — so toggling the rule moved every figure
   * below it down the page. The band follows the figure; it does not decide it.
   *
   * Done by hiding the band and asking the svg, not by unioning its children:
   * getBBox() on a child is in that child's OWN user space and excludes its own
   * transform, and the rows live inside a transformed group. Measuring
   * per-child got an answer that was wrong in a different way for every figure.
   */
  function fitBox(svg){
    var nf=svg.querySelector('g.nofit'), had=nf&&nf.style.display;
    if(nf) nf.style.display='none';
    var box;
    try{ box=svg.getBBox(); }finally{ if(nf) nf.style.display=had||''; }
    return box;
  }

  function refit(){
    document.querySelectorAll('.figure svg').forEach(function(svg){
      // Read the attribute every time and validate it before trusting it. The
      // first version cached getAttribute('viewBox') straight into dataset, so
      // any svg without one cached the string "null" and then wrote
      // "NaN undefined undefined NaN" back over its own viewBox.
      var raw = svg.dataset.vb || svg.getAttribute('viewBox');
      if(!raw) return;
      var base = String(raw).trim().split(/[ ,]+/).map(Number);
      if(base.length !== 4) return;
      for(var i=0;i<4;i++) if(!isFinite(base[i])) return;
      svg.dataset.vb = raw;
      var box;
      try{ box = fitBox(svg); }catch(e){ return; }
      if(!box || !isFinite(box.height) || !isFinite(box.y)) return;
      if(box.width === 0 && box.height === 0) return;   // hidden: not measurable
      var pad = 3;
      var top = box.y - pad;
      var height = Math.max(1, box.height + pad * 2);
      svg.setAttribute('viewBox', base[0]+' '+top.toFixed(1)+' '+base[2]+' '+height.toFixed(1));
    });
  }


  // Sliders write the same state the swatches do, so reset clears them too.
  var SL=${JSON.stringify(SLIDERS)};
  document.querySelectorAll('#side .tgb').forEach(function(b){
    b.addEventListener('click',function(){
      var on=!b.classList.contains('on');
      b.classList.toggle('on',on);
      b.textContent=on?'on':'off';
      set(b.dataset.tg, on?null:b.dataset.off);
      apply();
    });
  });
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

  // ---- feature toggles -------------------------------------------------
  // These change what is DRAWN, so the page carries a variant per combination
  // and this picks one. Everything else in the panel is a CSS variable.
  var FKEY='tds.feat.v1', feat={trim:true, compress:true, compressCompute:false, multiple:${MULT_DEFAULT}};
  try{ feat=Object.assign(feat, JSON.parse(localStorage.getItem(FKEY)||'{}')); }catch(e){}
  function featKey(){ return (feat.trim?'1':'0')+(feat.compress?'1':'0'); }
  /**
   * Morph between two feature variants.
   *
   * Both variants are already in the DOM, one of them display:none, and the
   * elements correspond because the same code drew them. So: read the outgoing
   * geometry, swap which variant is shown, and animate the incoming one from
   * where the outgoing one was. What you see is the bars moving between the two
   * layouts rather than the page cutting between them.
   *
   * Read from ATTRIBUTES, not from getBoundingClientRect: the incoming variant
   * is hidden when it is measured, and a hidden element has no box.
   *
   * Driven by element.animate() rather than a CSS transition. A transition
   * needs a previous computed value to move from, and an element that was
   * display:none until this frame has none — so it arrived at its final
   * position instantly, every time.
   *
   * Additive by design: if any of it fails the toggle still works, it just
   * happens at once.
   */
  var MORPH={duration:520, easing:'cubic-bezier(.33,1,.68,1)', fill:'none'};
  function geomOf(el){
    return {x:el.getAttribute('x'), cx:el.getAttribute('cx'),
            w:el.getAttribute('width'),
            vw:(el.style&&el.style.getPropertyValue('--w'))||''};
  }
  /**
   * Where everything was, keyed by what it IS.
   *
   * Matching by position in the document is what made the toggle chaotic:
   * switching compression adds bands and splits bars, so the nth element of
   * one drawing is not the nth of the next, and every pair after the first
   * difference animated something towards a place it had never been. Each bar,
   * mark and run-row slice carries a data-k -- its row, its kind, and which
   * one of that kind -- so the same thing is matched to itself.
   *
   * Everything without a key is left alone. Labels, ticks and band scrims
   * change wholesale between the two drawings and have no counterpart to
   * travel from; sliding them was most of the noise.
   */
  function snapGeom(root){
    var out={};
    root.querySelectorAll('[data-k]').forEach(function(el){
      out[el.getAttribute('data-k')]=geomOf(el);
    });
    return out;
  }
  /** Animate a freshly drawn figure out of the geometry the old one had. */
  function morphInto(root, snap){
    root.querySelectorAll('[data-k]').forEach(function(el){
      var g=snap[el.getAttribute('data-k')];
      try{
        // Nothing it was before: new to this drawing, so it arrives where it
        // belongs rather than travelling from somewhere it never was.
        if(!g){ el.animate([{opacity:0},{opacity:1}], MORPH); return; }
        var a={}, z={}, moved=false, now=geomOf(el);
        if(g.x!=null  && now.x!=null  && g.x!==now.x)   { a.x=g.x+'px';    z.x=now.x+'px';    moved=true; }
        if(g.cx!=null && now.cx!=null && g.cx!==now.cx) { a.cx=g.cx+'px';  z.cx=now.cx+'px';  moved=true; }
        if(g.vw && now.vw && g.vw!==now.vw)             { a['--w']=g.vw;   z['--w']=now.vw;   moved=true; }
        else if(g.w!=null && now.w!=null && g.w!==now.w){ a.width=g.w+'px';z.width=now.w+'px';moved=true; }
        if(moved) el.animate([a,z], MORPH);
      }catch(e){}
    });
  }

  function applyFeat(animate){
    document.documentElement.setAttribute('data-feat', featKey());
    /**
     * Read where everything IS before the figures are redrawn.
     *
     * The component replaces each figure's whole subtree, so the outgoing
     * elements are gone by the time the new ones exist -- there is nothing left
     * to measure afterwards. Snapshot first, redraw, then animate the new
     * elements out of the old geometry.
     */
    var snaps=null;
    if(animate && document.body.animate){
      snaps=[];
      document.querySelectorAll('svg[data-fig]').forEach(function(svg){
        snaps.push([svg.parentNode, snapGeom(svg)]);
      });
    }
    // The features are properties of a drawing. Every figure re-renders from
    // the events it was drawn from, through the same component.
    if(window.__renderFigures) window.__renderFigures({trim:feat.trim, compress:feat.compress,
      compressCompute:feat.compressCompute, multiple:feat.multiple});
    document.querySelectorAll('#side [data-feat]').forEach(function(b){
      var on=!!feat[b.dataset.feat];
      b.classList.toggle('on',on); b.textContent=on?'on':'off';
    });
    if(!snaps){ refit(); requestAnimationFrame(refit); return; }
    // A frame later, because the render is batched and the new subtree does not
    // exist yet. Refit first: it settles the viewBox, and an animation started
    // against a viewBox that then changes is an animation that drifts.
    requestAnimationFrame(function(){
      refit();
      snaps.forEach(function(p){
        var svg=p[0] && p[0].querySelector && p[0].querySelector('svg[data-fig]');
        if(svg) morphInto(svg, p[1]);
      });
      setTimeout(refit, MORPH.duration+40);
    });
  }
  var multOut=document.querySelector('[data-out="mult"]'),
      multIn=document.querySelector('input[data-mult]');
  if(multIn){
    multIn.value=feat.multiple;
    if(multOut) multOut.textContent=feat.multiple+'\u00d7';
    multIn.addEventListener('input',function(){
      feat.multiple=+multIn.value;
      if(multOut) multOut.textContent=feat.multiple+'\u00d7';
      try{ localStorage.setItem(FKEY,JSON.stringify(feat)); }catch(e){}
      applyFeat(true);
    });
  }
  document.querySelectorAll('#side [data-feat]').forEach(function(b){
    b.addEventListener('click',function(){
      feat[b.dataset.feat]=!feat[b.dataset.feat];
      try{ localStorage.setItem(FKEY,JSON.stringify(feat)); }catch(e){}
      applyFeat(true);
    });
  });

  applyFeat();
  // Text metrics change when the display font arrives, and a figure is sized
  // from its bbox — so the first fit is measured against the fallback and every
  // later one against the real font. That was the last 0.5px of drift between a
  // figure's height before and after a toggle.
  if(document.fonts && document.fonts.ready) document.fonts.ready.then(function(){ refit(); });

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
