const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,arrow,causal,wire,tag,ribbon,C,W,LBL,PLOT} from './micro.mjs';
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
  [{l:'at rest',svg:fig([{n:'a',at:[['queued',0],['started',10],['ok',44]],note:'discovered'},
    {n:'b',at:[['queued',44],['started',56],['ok',86]],note:'discovered'}],'','sequential')},
   {l:'hovering b',svg:fig([{n:'a',at:[['queued',0],['started',10],['ok',44]],dim:DIM},
    {n:'b',at:[['queued',44],['started',56],['ok',86]],note:'discovered',sel:true}],
    '','sequential hovered')}]);

D('c1','One request made three steps. The ribbon threads their queue circles at rest, so hovering a member draws <em>no</em> cable &mdash; the relationship is already on screen, and nothing preceded this request to cable back to.',
  // Hovering `b` lights what caused it and nothing else: `b` entire, the ribbon,
  // and on `a`'s row only the discovery bar, the queued mark that opens it and
  // the planned mark the ribbon threads. `a`'s own work had nothing to do with
  // `b` starting, so it stays faded.
  pair([{n:'req + a',at:[['queued',0],['started',5.6,'disc'],['ok',16],['planned',16],['started',22],['ok',66]],reported:1,note:'44ms'},
        {n:'b',at:[['planned',16],['started',22],['ok',74]],note:'52ms'},
        {n:'c',at:[['planned',16],['started',22],['ok',60]],note:'38ms'}],
    {arrows:[],
     focus:1,label:'fan-out',hoverNote:'hovering b'}));

D('c2','The whole shape: one request fans out to three steps, all three completing causes the next request, and that request produced a single step so it rolls into <code>d</code>&rsquo;s row. The ribbon covers the fan-out at rest; the cables only appear when you ask, and only for the hop the ribbon cannot span.',
  pair([{n:'req + a',at:[['queued',0],['started',4.2,'disc'],['ok',12],['planned',12],['started',14],['ok',34]],reported:1,note:'20ms'},
        {n:'b',at:[['planned',12],['started',14],['ok',46]],note:'32ms'},
        {n:'c',at:[['planned',12],['started',14],['ok',40]],note:'26ms'},
        {n:'d',at:[['queued',46],['started',56],['ok',74]],note:'18ms'}],
    {focus:3,

     label:'fan-out then coalesce',hoverNote:'d selected'}));

D('c3','Ribbons nest, so depth reads without indentation. Hovering <code>a1</code> lights <code>a1</code>, its ribbon, and the one thing that caused it &mdash; <code>a</code> resolving. Its sibling did not cause it and stays faded.',
  pair([{n:'req + a',at:[['queued',0],['started',3.5,'disc'],['ok',10],['planned',10],['started',12],['ok',32]],reported:1},
        {n:'b',at:[['planned',10],['started',12],['ok',36]]},
        {n:'req + a1',at:[['queued',34],['started',36.8,'disc'],['ok',42],['planned',42],['started',44],['ok',62]],reported:1},
        {n:'a2',at:[['planned',42],['started',44],['ok',66]]},
        {n:'req + b1',at:[['queued',38],['started',40.8,'disc'],['ok',46],['queued',46],['started',48],['ok',74]],reported:1}],
    {focus:2,
     label:'nested fan-out',hoverNote:'hovering a1'}));

D('c4','The ribbon covers exactly the members, so an uneven fan-out is legible at rest &mdash; you can see it produced three without counting. Hovering adds nothing here, because the ribbon has already said it.',
  // Every member of a fan-out is enqueued by the request and waits its turn, so
  // each carries the same queue interval and the planned mark that opens it.
  // Without them the ribbon threaded rows that had no circles to thread.
  pair([{n:'req + a',at:[['queued',0],['started',4.2,'disc'],['ok',12],['planned',12],['started',14],['ok',44]],reported:1},
        {n:'b',at:[['planned',12],['started',14],['ok',72]]},
        {n:'c',at:[['planned',12],['started',14],['ok',26]]},
        {n:'unrelated',at:[['started',20],['ok',60]]}],
    {arrows:[],focus:0,
     label:'unbalanced fan-out',hoverNote:'the request selected'}));

D('c5','Nothing static predicted the width. The ribbon reports what happened rather than promising a shape, and the one cable worth drawing is the hop from <code>decide</code> into the request &mdash; not the request out to its own members.',
  pair([{n:'decide',at:[['started',0],['ok',18]]},
        {n:'req + d1',at:[['queued',20],['started',23.5,'disc'],['ok',30],['planned',30],['started',32],['ok',58]],reported:1},
        {n:'d2',at:[['planned',30],['started',32],['ok',62]]},{n:'d3',at:[['planned',30],['started',32],['ok',54]]},{n:'d4',at:[['planned',30],['started',32],['ok',66]]}],
    {focus:1,
     label:'dynamic width',hoverNote:'the request selected'}));

D('c6','The ribbon covers both steps the resumption discovered, so nothing implies an order between them. The cable shows only what caused the resumption.',
  pair([{n:'a',at:[['started',2],['ok',28]]},
        {n:'req + b',at:[['queued',32],['started',36.9,'disc'],['ok',46],['planned',46],['started',48],['ok',76]],reported:1},
        {n:'c',at:[['planned',46],['started',48],['ok',68]]}],
    {focus:1,
     label:'one resumption, two steps',hoverNote:'the request selected'}));

D('c7','Hovering one branch is the check: its arrow stays inside the branch. A crossed pairing would be visible as a crossed arrow.',
  pair([{n:'left-1',at:[['started',4],['ok',28]]},{n:'right-1',at:[['started',4],['ok',34]]},
        {n:'left-2',at:[['queued',30],['started',42],['ok',64]]},
        {n:'right-2',at:[['queued',36],['started',48],['ok',74]]}],
    {focus:2,label:'branch membership',hoverNote:'hovering left-2'}));

D('c8','Each branch scheduled its own request, so there are two. Hovering shows one feeder, not two.',
  pair([{n:'req + a',at:[['queued',0],['started',2,'disc'],['ok',6],['planned',6],['started',8],['ok',26]],reported:1},
        {n:'b',at:[['planned',6],['started',8],['ok',32]]},
        {n:'req + a2',at:[['queued',26],['started',28,'disc'],['ok',32],['queued',32],['started',34],['ok',54]],reported:1},
        {n:'req + b2',at:[['queued',32],['started',34,'disc'],['ok',38],['queued',38],['started',40],['ok',62]],reported:1}],
    {focus:2,
     label:'no coalescing',hoverNote:'hovering a2'}));

D('c9','The winner is a dependency and gets a solid arrow; the losers were alternates and get dashed ones. Both only appear once you ask.',
  pair([{n:'fast',at:[['started',4],['ok',24]]},{n:'slow',at:[['started',4],['ok',48]]},
        {n:'next',at:[['queued',26],['started',38],['ok',62]]}],
    {focus:2,
     hoverExtra:`<path d="M${px(48)} ${cy(1)} C ${px(56)} ${cy(1)}, ${px(30)} ${cy(2)}, ${px(40)} ${cy(2)}" fill="none" stroke="${C.acc}" stroke-width="1.2" stroke-dasharray="2.5 2.5" opacity=".6"/>`,
     label:'race',hoverNote:'the request selected'}));

D('c10','A dead end is drawn by absence. Hovering it lights its own row and nothing downstream, because there is nothing downstream.',
  pair([{n:'a',at:[['started',4],['ok',30]]},{n:'orphan',at:[['started',4],['ok',38]]},
        {n:'b',at:[['queued',32],['started',44],['ok',66]]}],
    {arrows:[],focus:1,label:'dead end',hoverNote:'hovering orphan, no outgoing arrow'}));

D('c12','Rows sort by start, so plan order is not row order. The ribbon carries the plan and the axis carries the time; neither has to lie, and no cable is needed to repeat it.',
  pair([{n:'req + b',at:[['queued',0],['started',4.2,'disc'],['ok',12],['planned',12],['started',14],['ok',34]],reported:1,note:'planned 2nd'},
        {n:'a',at:[['planned',12],['started',18],['ok',58]],note:'planned 1st'},{n:'c',at:[['planned',12],['started',22],['ok',38]],note:'planned 3rd'}],
    {arrows:[],focus:0,
     label:'plan order vs row order',hoverNote:'the request selected'}));

D('c13','Response order stopped reflecting structure, so row order is time order and the arrow is the only structural claim — and only when asked for.',
  pair([{n:'early',at:[['started',2],['ok',16]]},
        {n:'late',at:[['queued',18],['started',74],['ok',84]],note:'+249ms planning'},
        {n:'other',at:[['started',74],['ok',88]]}],
    {focus:1,label:'discovery after unrelated awaits',hoverNote:'hovering late'}));

fs.writeFileSync(HERE+'items-a.json',JSON.stringify(E));
console.log('items',Object.keys(E).length,'frames',Object.values(E).reduce((n,x)=>n+x.frames.length,0));
