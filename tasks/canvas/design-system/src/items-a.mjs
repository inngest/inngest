const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,arrow,causal,wire,tag,ribbon,C,W,LBL,PLOT} from './micro.mjs';
const E={};
const DIM=.15;

/** Same rows twice: at rest, then with everything off the path faded. */
function pair(rows,{lit,arrows=[],restExtra='',hoverExtra='',label='',hoverNote='hovered',rib,ribs}){
  const all=ribs||(rib?[rib]:[]);
  const under=all.map(r=>ribbon(r.x,r.rows.map(i=>cy(i)))).join('');
  const marked=rows.map((r,i)=>{
    const on=all.find(g=>g.rows.includes(i));
    return on?{...r,noHalo:[on.x]}:r;
  });
  const rest=fig(marked,restExtra,label+' at rest',under);
  const dimmed=marked.map((r,i)=>lit.includes(i)?r:{...r,dim:DIM});
  const focus = /^hovering/.test(hoverNote) ? lit[lit.length-1] : null;
  const hov=fig(dimmed.map((r,i)=>i===focus?{...r,sel:true}:r),
    arrows.map(a=>Array.isArray(a)?arrow(...a):a).join('')+hoverExtra,label+' hovered',
    all.map(r=>ribbon(r.x,r.rows.map(i=>cy(i)),{o:.35})).join(''));
  return [{l:'at rest',svg:rest},{l:hoverNote,svg:hov}];
}
const D=(k,d,frames)=>{E[k]={d,frames};};

D('c0','In a single sequential thread the SDK answers and runs in the same execution &mdash; one HTTP call, not two &mdash; so there is no discovery bar. But the execution still <em>began</em> not knowing what it would run, and the blue start circle says so. A step planned by an earlier request opens on the grey mark instead.',
  [{l:'at rest',svg:fig([{n:'a',segs:[['idle',0,10],['good',10,34]],note:'discovered'},
    {n:'b',segs:[['idle',48,8],['good',56,30]],note:'discovered'}],'','sequential')},
   {l:'hovering b',svg:fig([{n:'a',segs:[['idle',0,10],['good',10,34]],dim:DIM},
    {n:'b',segs:[['idle',48,8],['good',56,30]],note:'discovered',sel:true}],
    wire(px(44),cy(0),px(48),cy(1)),'sequential hovered')}]);

D('c1','One request made three steps. The ribbon threads their queue circles at rest, so hovering a member draws <em>no</em> cable &mdash; the relationship is already on screen, and nothing preceded this request to cable back to.',
  pair([{n:'req + a',segs:[['disc',0,16],['idle',16,6],['good',22,44]],note:'44ms'},
        {n:'b',segs:[['idle',16,6],['good',22,52]],note:'52ms'},
        {n:'c',segs:[['idle',16,6],['good',22,38]],note:'38ms'}],
    {lit:[0,1],arrows:[],label:'fan-out',hoverNote:'hovering b',rib:{x:16,rows:[0,1,2]}}));

D('c2','The whole shape: one request fans out to three steps, all three completing causes the next request, and that request produced a single step so it rolls into <code>d</code>&rsquo;s row. The ribbon covers the fan-out at rest; the cables only appear when you ask, and only for the hop the ribbon cannot span.',
  pair([{n:'req + a',segs:[['disc',0,12],['idle',12,2],['good',14,20]],note:'20ms'},
        {n:'b',segs:[['idle',12,2],['good',14,32]],note:'32ms'},
        {n:'c',segs:[['idle',12,2],['good',14,26]],note:'26ms'},
        {n:'d',segs:[['idle',46,10],['good',56,18]],note:'18ms'}],
    {lit:[0,1,2,3],
     arrows:[wire(px(34),cy(0),px(46),cy(3),{i:0})+wire(px(46),cy(1),px(46),cy(3),{i:2})+wire(px(40),cy(2),px(46),cy(3),{i:1})],
     label:'fan-out then coalesce',hoverNote:'d selected',rib:{x:12,rows:[0,1,2]}}));

D('c3','Chevrons nest naturally, so depth reads without indentation. Hovering one level lights only that level.',
  pair([{n:'req + a',segs:[['disc',0,10],['idle',10,2],['good',12,20]]},
        {n:'b',segs:[['idle',10,2],['good',12,24]]},
        {n:'req + a1',segs:[['disc',34,8],['idle',42,2],['good',44,18]]},
        {n:'a2',segs:[['idle',42,2],['good',44,22]]},
        {n:'req + b1',segs:[['disc',38,8],['idle',46,2],['good',48,26]]}],
    {lit:[0,2,3],arrows:[wire(px(32),cy(0),px(34),cy(2))],
     label:'nested fan-out',hoverNote:'hovering a1',ribs:[{x:10,rows:[0,1]},{x:42,rows:[2,3]}]}));

D('c4','The ribbon covers exactly the members, so an uneven fan-out is legible at rest &mdash; you can see it produced three without counting. Hovering adds nothing here, because the ribbon has already said it.',
  pair([{n:'req + a',segs:[['disc',0,12],['good',14,30]],},
        {n:'b',segs:[['good',14,58]]},{n:'c',segs:[['good',14,12]]},{n:'unrelated',segs:[['good',20,40]]}],
    {lit:[0,1,2],arrows:[],
     label:'unbalanced fan-out',hoverNote:'the request selected',rib:{x:12,rows:[0,1,2]}}));

D('c5','Nothing static predicted the width. The ribbon reports what happened rather than promising a shape, and the one cable worth drawing is the hop from <code>decide</code> into the request &mdash; not the request out to its own members.',
  pair([{n:'decide',segs:[['good',0,18]]},
        {n:'req + d1',segs:[['disc',20,10],['idle',30,2],['good',32,26]]},
        {n:'d2',segs:[['idle',30,2],['good',32,30]]},{n:'d3',segs:[['idle',30,2],['good',32,22]]},{n:'d4',segs:[['idle',30,2],['good',32,34]]}],
    {lit:[0,1,2,3,4],arrows:[wire(px(18),cy(0),px(20),cy(1))],
     label:'dynamic width',hoverNote:'the request selected',rib:{x:30,rows:[1,2,3,4]}}));

D('c6','The ribbon covers both steps the resumption discovered, so nothing implies an order between them. The cable shows only what caused the resumption.',
  pair([{n:'a',segs:[['good',2,26]]},
        {n:'req + b',segs:[['disc',32,14],['idle',46,2],['good',48,28]]},
        {n:'c',segs:[['idle',46,2],['good',48,20]]}],
    {lit:[0,1,2],arrows:[wire(px(28),cy(0),px(32),cy(1))],
     label:'one resumption, two steps',hoverNote:'the request selected',rib:{x:46,rows:[1,2]}}));

D('c7','Hovering one branch is the check: its arrow stays inside the branch. A crossed pairing would be visible as a crossed arrow.',
  pair([{n:'left-1',segs:[['good',4,24]]},{n:'right-1',segs:[['good',4,30]]},
        {n:'left-2',segs:[['idle',30,10],['good',42,22]]},
        {n:'right-2',segs:[['idle',36,10],['good',48,26]]}],
    {lit:[0,2],arrows:[wire(px(28),cy(0),px(30),cy(2))],label:'branch membership',hoverNote:'hovering left-2'}));

D('c8','Each branch scheduled its own request, so there are two. Hovering shows one feeder, not two.',
  pair([{n:'a',segs:[['good',2,24]]},{n:'b',segs:[['good',2,30]]},
        {n:'a2',segs:[['idle',28,10],['good',40,20]]},
        {n:'b2',segs:[['idle',34,10],['good',46,22]]}],
    {lit:[0,2],arrows:[wire(px(26),cy(0),px(28),cy(2))],label:'no coalescing',hoverNote:'hovering a2'}));

D('c9','The winner is a dependency and gets a solid arrow; the losers were alternates and get dashed ones. Both only appear once you ask.',
  pair([{n:'fast',segs:[['good',4,20]]},{n:'slow',segs:[['good',4,44]]},
        {n:'next',segs:[['idle',26,10],['good',38,24]]}],
    {lit:[0,1,2],arrows:[wire(px(24),cy(0),px(26),cy(2))],
     hoverExtra:`<path d="M${px(48)} ${cy(1)} C ${px(56)} ${cy(1)}, ${px(30)} ${cy(2)}, ${px(40)} ${cy(2)}" fill="none" stroke="${C.acc}" stroke-width="1.2" stroke-dasharray="2.5 2.5" opacity=".6"/>`,
     label:'race',hoverNote:'the request selected'}));

D('c10','A dead end is drawn by absence. Hovering it lights its own row and nothing downstream, because there is nothing downstream.',
  pair([{n:'a',segs:[['good',4,26]]},{n:'orphan',segs:[['good',4,34]]},
        {n:'b',segs:[['idle',32,10],['good',44,22]]}],
    {lit:[1],arrows:[],label:'dead end',hoverNote:'hovering orphan, no outgoing arrow'}));

D('c11','Inferred grouping gets a dashed ribbon and a dashed cable, so a guess never looks like a reported fact.',
  [{l:'at rest',svg:fig([{n:'req + a',segs:[['disc',0,12],['good',14,30]]},{n:'b',segs:[['good',14,26]]}],
    `<path d="M${px(12)+7.5} ${cy(0)-3.6} L${px(12)+11.9} ${cy(0)} L${px(12)+7.5} ${cy(0)+3.6}" fill="none" stroke="${C.acc}" stroke-width="1.5" stroke-dasharray="2 1.6"/>`+
    tag(px(26),cy(0)-5,'inferred',C.acc),'inferred at rest')},
   {l:'selected',svg:fig([{n:'req + a',segs:[['disc',0,12],['good',14,30]],sel:true},{n:'b',segs:[['good',14,26]]}],
    `<path d="M${px(12)} ${cy(0)} C ${px(16)} ${cy(0)}, ${px(10)} ${cy(1)}, ${px(11)} ${cy(1)}" fill="none" stroke="${C.acc}" stroke-width="1.3" stroke-dasharray="2.5 2.5"/>`+
    `<path d="M${px(11)} ${cy(1)-2.2} L${px(14)} ${cy(1)} L${px(11)} ${cy(1)+2.2} Z" fill="${C.acc}" opacity=".7"/>`+
    tag(px(26),cy(1)+9,'grouping inferred from overlap',C.mut),'inferred selected')}]);

D('c12','Rows sort by start, so plan order is not row order. The ribbon carries the plan and the axis carries the time; neither has to lie, and no cable is needed to repeat it.',
  pair([{n:'req + b',segs:[['disc',0,12],['idle',12,2],['good',14,20]],note:'planned 2nd'},
        {n:'a',segs:[['idle',12,6],['good',18,40]],note:'planned 1st'},{n:'c',segs:[['idle',12,10],['good',22,16]],note:'planned 3rd'}],
    {lit:[0,1,2],arrows:[],
     label:'plan order vs row order',hoverNote:'the request selected',rib:{x:12,rows:[0,1,2]}}));

D('c13','Response order stopped reflecting structure, so row order is time order and the arrow is the only structural claim — and only when asked for.',
  pair([{n:'early',segs:[['good',2,14]]},
        {n:'late',segs:[['idle',18,56.0],['good',74,10]],note:'+249ms planning'},
        {n:'other',segs:[['good',74,14]]}],
    {lit:[0,1],arrows:[wire(px(16),cy(0),px(18),cy(1))],label:'discovery after unrelated awaits',hoverNote:'hovering late'}));

fs.writeFileSync(HERE+'items-a.json',JSON.stringify(E));
console.log('items',Object.keys(E).length,'frames',Object.values(E).reduce((n,x)=>n+x.frames.length,0));
