const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,fin,px,cy,arrow,causal,wire,tag,ribbon,C,W,LBL,PLOT} from './micro.mjs';
import * as R from './rules.mjs';
import {resolveRow} from './micro.mjs';
import {setFrame} from './micro.mjs'; setFrame(true);
const E={};
const DIM=.15;

/** Same rows twice: at rest, then with everything off the path faded. */
/**
 * `lit` names what stays at full opacity in the hovered frame. An entry is
 * either a row index (the whole row) or `{row, bars, dots}` — the parts of a
 * row that caused the hovered one. A row that merely *contains* the cause is
 * not itself the subject: hovering `b` lights `a`'s discovery bar and its queue
 * circle, and leaves `a`'s `ok` bar faded with everything else.
 *
 * `focus` names the row that takes the selection wash. It defaults to the last
 * fully-lit row, which is right when the frame is described as "hovering x" and
 * wrong when it is "the request selected", so those pass it explicitly.
 */
function pair(rows,{arrows=[],restExtra='',hoverExtra='',label='',hoverNote='hovered',focus,frame}){
  const marked=rows;
  const rest=fig(marked,restExtra,label+' at rest','',{frame});
  // What survives the hover is derived from what caused the row -- see
  // rules.attention. It used to be listed per figure, bar positions and all.
  if(focus==null) throw new Error('pair("'+label+'"): needs `focus`, the row the hover is about');
  const focusRow=focus;
  const spec=R.attention(rows.map(resolveRow), focusRow);
  const dimmed=marked.map((r,i)=>{
    const p=spec.get(i);
    if(p===true) return r;
    if(!p) return {...r,dim:DIM};
    return {...r,dim:DIM,lit:p.bars,litDots:p.dots};
  });
  // The ribbon fades with everything else that is not the answer to the hover.
  const hov=fig(dimmed.map((r,i)=>i===focusRow?{...r,sel:true}:r),
    hoverExtra,label+' hovered','',{frame,ribO:R.FOCUS.ribbonDim,focus:focusRow});
  return [{l:'at rest',svg:rest},{l:hoverNote,svg:hov}];
}
const D=(k,d,frames)=>{E[k]={d,frames};};

D('c0','In a single sequential thread the SDK answers and runs in the same execution &mdash; one HTTP call, not two &mdash; so there is no discovery bar. But the execution still <em>began</em> not knowing what it would run, and the blue start circle says so. A step planned by an earlier request opens on the grey mark instead.',
  // `b`'s queue begins the instant `a` resolves. There is no gap to draw,
  // because there is no second request to wait for — which is the whole point
  // of the figure, and a gap here quietly contradicted it.
  [{l:'at rest',svg:fig([{n:'a',at:[['queued',0],['started',100],['ok',440]],note:'discovered'},
    {n:'b',at:[['queued',440],['started',560],['ok',860]],note:'discovered'},
    fin(860,110,70)],'','sequential')},
   {l:'hovering b',svg:fig([{n:'a',at:[['queued',0],['started',100],['ok',440]],dim:DIM},
    {n:'b',at:[['queued',440],['started',560],['ok',860]],note:'discovered',sel:true},
    {...fin(860,110,70),dim:DIM}],
    '','sequential hovered')}]);

D('c1','One request made three steps. The ribbon threads their queue circles at rest, so hovering a member draws <em>no</em> cable &mdash; the relationship is already on screen, and nothing preceded this request to cable back to.',
  // Hovering `b` lights what caused it and nothing else: `b` entire, the ribbon,
  // and on `a`'s row only the discovery bar, the queued mark that opens it and
  // the planned mark the ribbon threads. `a`'s own work had nothing to do with
  // `b` starting, so it stays faded.
  pair([{n:'req + a',at:[['queued',0],['started',56,'disc'],['ok',160],['planned',160],['started',220],['ok',660]],reported:1,note:'44ms'},
        {n:'b',at:[['planned',160],['started',220],['ok',740]],note:'52ms'},
        {n:'c',at:[['planned',160],['started',220],['ok',600]],note:'38ms'},
        fin(740,60,100)],
    {arrows:[],
     focus:1,label:'fan-out',hoverNote:'hovering b'}));

D('c2','The whole shape: one request fans out to three steps, all three completing causes the next request, and that request produced a single step so it rolls into <code>d</code>&rsquo;s row. The ribbon covers the fan-out at rest; the cables only appear when you ask, and only for the hop the ribbon cannot span.',
  pair([{n:'req + a',at:[['queued',0],['started',42,'disc'],['ok',120],['planned',120],['started',140],['ok',340]],reported:1,note:'20ms'},
        {n:'b',at:[['planned',120],['started',140],['ok',460]],note:'32ms'},
        {n:'c',at:[['planned',120],['started',140],['ok',400]],note:'26ms'},
        {n:'d',at:[['queued',460],['started',560],['ok',740]],note:'18ms'},
        fin(740,90,80)],
    {focus:3,

     label:'fan-out then coalesce',hoverNote:'d selected'}));

D('c3','Ribbons nest, so depth reads without indentation. Hovering <code>a1</code> lights <code>a1</code>, its ribbon, and the one thing that caused it &mdash; <code>a</code> resolving. Its sibling did not cause it and stays faded.',
  pair([{n:'req + a',at:[['queued',0],['started',35,'disc'],['ok',100],['planned',100],['started',120],['ok',320]],reported:1},
        {n:'b',at:[['planned',100],['started',120],['ok',360]]},
        {n:'req + a1',at:[['queued',340],['started',368,'disc'],['ok',420],['planned',420],['started',440],['ok',620]],reported:1},
        {n:'a2',at:[['planned',420],['started',440],['ok',660]]},
        {n:'req + b1',at:[['queued',380],['started',408,'disc'],['ok',460],['queued',460],['started',480],['ok',740]],reported:1},
        fin(740,30,55)],
    {focus:2,
     label:'nested fan-out',hoverNote:'hovering a1'}));

D('c4','The ribbon covers exactly the members, so an uneven fan-out is legible at rest &mdash; you can see it produced three without counting. Hovering adds nothing here, because the ribbon has already said it.',
  // Every member of a fan-out is enqueued by the request and waits its turn, so
  // each carries the same queue interval and the planned mark that opens it.
  // Without them the ribbon threaded rows that had no circles to thread.
  pair([{n:'req + a',at:[['queued',0],['started',42,'disc'],['ok',120],['planned',120],['started',140],['ok',440]],reported:1},
        {n:'b',at:[['planned',120],['started',140],['ok',720]]},
        {n:'c',at:[['planned',120],['started',140],['ok',260]]},
        {n:'unrelated',at:[['started',200],['ok',600]]},
        fin(720,40,78)],
    {arrows:[],focus:0,
     label:'unbalanced fan-out',hoverNote:'the request selected'}));

D('c5','Nothing static predicted the width. The ribbon reports what happened rather than promising a shape, and the one cable worth drawing is the hop from <code>decide</code> into the request &mdash; not the request out to its own members.',
  pair([{n:'decide',at:[['started',0],['ok',180]]},
        {n:'req + d1',at:[['queued',200],['started',235,'disc'],['ok',300],['planned',300],['started',320],['ok',580]],reported:1},
        {n:'d2',at:[['planned',300],['started',320],['ok',620]]},{n:'d3',at:[['planned',300],['started',320],['ok',540]]},{n:'d4',at:[['planned',300],['started',320],['ok',660]]},
        fin(660,35,65)],
    {focus:1,
     label:'dynamic width',hoverNote:'the request selected'}));

D('c6','The ribbon covers both steps the resumption discovered, so nothing implies an order between them. The cable shows only what caused the resumption.',
  pair([{n:'a',at:[['started',20],['ok',280]]},
        {n:'req + b',at:[['queued',320],['started',369,'disc'],['ok',460],['planned',460],['started',480],['ok',760]],reported:1},
        {n:'c',at:[['planned',460],['started',480],['ok',680]]},
        fin(760,50,90)],
    {focus:1,
     label:'one resumption, two steps',hoverNote:'the request selected'}));

D('c7','Hovering one branch is the check: its arrow stays inside the branch. A crossed pairing would be visible as a crossed arrow.',
  pair([{n:'left-1',at:[['started',40],['ok',280]]},{n:'right-1',at:[['started',40],['ok',340]]},
        {n:'left-2',at:[['queued',300],['started',420],['ok',640]]},
        {n:'right-2',at:[['queued',360],['started',480],['ok',740]]},
        fin(740,120,80)],
    {focus:2,label:'branch membership',hoverNote:'hovering left-2'}));

D('c8','Each branch scheduled its own request, so there are two. Hovering shows one feeder, not two.',
  pair([{n:'req + a',at:[['queued',0],['started',20,'disc'],['ok',60],['planned',60],['started',80],['ok',260]],reported:1},
        {n:'b',at:[['planned',60],['started',80],['ok',320]]},
        {n:'req + a2',at:[['queued',260],['started',280,'disc'],['ok',320],['queued',320],['started',340],['ok',540]],reported:1},
        {n:'req + b2',at:[['queued',320],['started',340,'disc'],['ok',380],['queued',380],['started',400],['ok',620]],reported:1},
        fin(620,20,40)],
    {focus:2,
     label:'no coalescing',hoverNote:'hovering a2'}));

D('c9','The winner resolved before the next request was queued, so it caused it and gets a cable. The loser resolved after that request had already started &mdash; it cannot have caused it, so it gets nothing. Drawing the loser as a faded alternate was the trace guessing at an intent the spans do not carry.',
  pair([{n:'fast',at:[['started',40],['ok',240]]},{n:'slow',at:[['started',40],['ok',480]]},
        {n:'next',at:[['queued',260],['started',380],['ok',620]]},
        fin(620,120,80)],
    {focus:2,label:'race',hoverNote:'the request selected'}));

D('c10','A dead end is drawn by absence. Hovering it lights its own row and nothing downstream, because there is nothing downstream.',
  pair([{n:'a',at:[['started',40],['ok',300]]},{n:'orphan',at:[['started',40],['ok',380]]},
        {n:'b',at:[['queued',320],['started',440],['ok',660]]},
        fin(660,120,80)],
    {arrows:[],focus:1,label:'dead end',hoverNote:'hovering orphan, no outgoing arrow'}));

D('c12','Rows sort by start, so plan order is not row order. The ribbon carries the plan and the axis carries the time; neither has to lie, and no cable is needed to repeat it.',
  pair([{n:'req + b',at:[['queued',0],['started',42,'disc'],['ok',120],['planned',120],['started',140],['ok',340]],reported:1,note:'planned 2nd'},
        {n:'a',at:[['planned',120],['started',180],['ok',580]],note:'planned 1st'},{n:'c',at:[['planned',120],['started',220],['ok',380]],note:'planned 3rd'},
        fin(580,60,78)],
    {arrows:[],focus:0,
     label:'plan order vs row order',hoverNote:'the request selected'}));

D('c13','Response order stopped reflecting structure, so row order is time order and the arrow is the only structural claim — and only when asked for.',
  pair([{n:'early',at:[['started',20],['ok',160]]},
        {n:'late',at:[['queued',180],['started',740],['ok',840]],note:'+249ms planning'},
        {n:'other',at:[['started',740],['ok',880]]},
        fin(880,60,90)],
    {focus:1,label:'discovery after unrelated awaits',hoverNote:'hovering late'}));

fs.writeFileSync(HERE+'items-a.json',JSON.stringify(E));
console.log('items',Object.keys(E).length,'frames',Object.values(E).reduce((n,x)=>n+x.frames.length,0));
