const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,arrow,tag,C,W,LBL,PLOT} from './micro.mjs';
const E={};const DIM=.15;

// c67 — the request fails, is retried, and eventually produces the step.
E.c67={d:'The request that would produce <code>b</code> failed twice before it worked. It lands in <code>b</code>&rsquo;s row anyway, so the row tells the whole story of how <code>b</code> came to exist. The bar stays discovery blue &mdash; this was Inngest&rsquo;s failure, not the user&rsquo;s &mdash; and the hollow red circles carry the outcome.',
 frames:[
  {l:'at rest',svg:fig([
    {n:'a',segs:[['good',0,18]]},
    {n:'req + b',segs:[['disc!',22,8],['idle',30,10],['disc!',40,8],['idle',48,16],['disc',64,8],['idle',72,4],['good',76,20]],note:'2 failed requests  20ms'},
  ],tag(px(31),cy(1)+13,'backoff 1s',C.mut)+tag(px(50),cy(1)+13,'backoff 2s',C.mut),'discovery retried')},
  {l:'hovering b',svg:fig([
    {n:'a',segs:[['good',0,18]],dim:DIM},
    {n:'req + b',segs:[['disc!',22,8],['idle',30,10],['disc!',40,8],['idle',48,16],['disc',64,8],['idle',72,4],['good',76,20]],note:'20ms',sel:true},
  ],arrow(px(18),cy(0),px(22),cy(1)),'discovery retried, hovered')},
 ]};

// c68 — it never succeeds, so there is no step to host it.
E.c68={d:'Every attempt failed, so no step was ever produced and there is no row to attach to. This is the one case where a request gets a row of its own &mdash; and the only place a filled red circle appears on something that is not the user&rsquo;s code.',
 frames:[
  {l:'at rest',svg:fig([
    {n:'a',segs:[['good',0,18]]},
    {n:'discovery',segs:[['disc!',22,8],['idle',30,10],['disc!',40,8],['idle',48,18],['disc!',66,8]],note:'3 attempts  gave up'},
    {n:'Run',segs:[['idle',0,74],['bad',74,3]],dim:.6,note:'FAILED'},
  ],'','discovery exhausted')},
  {l:'hovering the request',svg:fig([
    {n:'a',segs:[['good',0,18]],dim:DIM},
    {n:'discovery',segs:[['disc!',22,8],['idle',30,10],['disc!',40,8],['idle',48,18],['disc!',66,8]],note:'gave up',sel:true},
    {n:'Run',segs:[['idle',0,74],['bad',74,3]],dim:DIM},
  ],arrow(px(18),cy(0),px(22),cy(1))+tag(px(76),cy(1)+2.5,'no step was ever produced',C.mut),'discovery exhausted, hovered')},
 ]};

fs.writeFileSync(HERE+'items-disc.json',JSON.stringify(E));
console.log('ok',Object.keys(E).join(' '));
