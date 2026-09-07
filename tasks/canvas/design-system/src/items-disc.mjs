const HERE=new URL('./',import.meta.url).pathname;
import fs from 'fs';
import {fig,px,cy,arrow,tag,C,W,LBL,PLOT} from './micro.mjs';
import {setFrame} from './micro.mjs'; setFrame(true);
const E={};const DIM=.15;

// c67 — the request fails, is retried, and eventually produces the step.
E.c67={d:'The request that would produce <code>b</code> failed twice before it worked. It lands in <code>b</code>&rsquo;s row anyway, so the row tells the whole story of how <code>b</code> came to exist. The bar stays discovery blue &mdash; this was Inngest&rsquo;s failure, not the user&rsquo;s &mdash; and the hollow red circles carry the outcome.',
 frames:[
  {l:'at rest',svg:fig([
    {n:'a',at:[['started',0],['ok',18]]},
    {n:'req + b',at:[['started',22],['retry',30],['started',40,'disc'],['retry',48],['started',64,'disc'],['ok',72],['queued',72],['started',76],['ok',96]],reported:1,note:'2 failed requests  20ms'},
  ],tag(px(31),cy(1)+13,'backoff 1s',C.mut)+tag(px(50),cy(1)+13,'backoff 2s',C.mut),'discovery retried')},
  {l:'hovering b',svg:fig([
    {n:'a',at:[['started',0],['ok',18]],dim:DIM},
    {n:'req + b',at:[['started',22],['retry',30],['started',40,'disc'],['retry',48],['started',64,'disc'],['ok',72],['queued',72],['started',76],['ok',96]],reported:1,note:'20ms',sel:true},
  ],arrow(px(18),cy(0),px(22),cy(1)),'discovery retried, hovered')},
 ]};

// c68 — it never succeeds, so there is no step to host it.
E.c68={d:'Every attempt failed, so no step was ever produced and there is no row to attach to. This is the one case where a request gets a row of its own &mdash; and the only place a filled red circle appears on something that is not the user&rsquo;s code.',
 frames:[
  {l:'at rest',svg:fig([
    {n:'a',at:[['started',0],['ok',18]]},
    {n:'discovery',at:[['started',22],['retry',30],['started',40],['retry',48],['started',66],['failed',74]],kind:'disc',note:'3 attempts  gave up'},
  ],'','discovery exhausted')},
  {l:'hovering the request',svg:fig([
    {n:'a',at:[['started',0],['ok',18]],dim:DIM},
    {n:'discovery',at:[['started',22],['retry',30],['started',40],['retry',48],['started',66],['failed',74]],kind:'disc',note:'gave up',sel:true},
  ],arrow(px(18),cy(0),px(22),cy(1))+tag(px(76),cy(1)+2.5,'no step was ever produced',C.mut),'discovery exhausted, hovered')},
 ]};

fs.writeFileSync(HERE+'items-disc.json',JSON.stringify(E));
console.log('ok',Object.keys(E).join(' '));
