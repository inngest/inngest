/**
 * The artifact, served and rebuilt on every save.
 *
 * `node serve.mjs` then open http://localhost:5199. Editing anything under
 * `src/` rebuilds `trace-design-system.html` and reloads the page — which
 * matters more here than in most places, because the whole method is "change a
 * rule, look at it, react", and a manual rebuild between those steps is enough
 * friction to stop it happening.
 *
 * The page keeps its scroll position and its open tab across a reload: those
 * are already in sessionStorage/localStorage for the tab, and the scroll is
 * restored here. Losing your place on a 1.8MB page after every keystroke makes
 * the loop worse, not better.
 */
import http from 'http';
import fs from 'fs';
import path from 'path';
import { execFile } from 'child_process';

const SRC = new URL('./', import.meta.url).pathname;
const OUT = path.join(SRC, '..', 'trace-design-system.html');
const PORT = +(process.env.PORT || 5199);

const clients = new Set();
const notify = msg => { for (const c of clients) c.write(`data: ${msg}\n\n`); };

let building = false, again = false;
function build(){
  if(building){ again = true; return; }
  building = true;
  const t0 = Date.now();
  execFile('node', [path.join(SRC,'ds.mjs')], {cwd: SRC}, (err, out, errOut) => {
    building = false;
    if(err){
      // Keep serving the last good build; the page shows the error instead of
      // silently going stale.
      const msg = (errOut || String(err)).split('\n').slice(0,12).join('\n');
      console.error('build failed\n' + msg);
      notify('error:' + JSON.stringify(msg));
    }else{
      console.log(`built in ${Date.now()-t0}ms — ${out.trim()}`);
      notify('reload');
    }
    if(again){ again = false; build(); }
  });
}

// Sources only. The build writes a .json per figure module back into this
// directory, so watching those would rebuild forever.
//
// Polled rather than fs.watch: recursive watching is unreliable here, where it
// fired once and then went quiet -- worse than no watcher at all, because the
// page goes stale while still looking live. Twenty-odd stats every 300ms costs
// nothing and cannot silently stop.
const mtimes = new Map();
function scan(){
  let changed = false;
  const seen = new Set();
  for(const name of fs.readdirSync(SRC)){
    if(!name.endsWith(".mjs")) continue;
    seen.add(name);
    let t; try{ t = fs.statSync(path.join(SRC,name)).mtimeMs; }catch{ continue; }
    if(mtimes.get(name) !== t){ if(mtimes.has(name)) changed = true; mtimes.set(name, t); }
  }
  for(const name of [...mtimes.keys()]) if(!seen.has(name)){ mtimes.delete(name); changed = true; }
  if(changed) build();
}
scan();                        // seed, so start-up does not look like a change
setInterval(scan, 300).unref();

const LIVE = `<script>
(function(){
  var k='ds.scroll';
  var y=sessionStorage.getItem(k);
  if(y) addEventListener('load',function(){ scrollTo(0,+y); });
  addEventListener('beforeunload',function(){ sessionStorage.setItem(k,scrollY); });
  var box;
  function show(t){
    if(!box){
      box=document.createElement('pre');
      box.style.cssText='position:fixed;left:0;right:0;bottom:0;z-index:9999;margin:0;'+
        'padding:10px 14px;background:#3b1111;color:#ffd7d7;font:12px/1.5 ui-monospace,monospace;'+
        'white-space:pre-wrap;max-height:44vh;overflow:auto;border-top:2px solid #a33';
      document.body.appendChild(box);
    }
    box.textContent=t;
  }
  var es=new EventSource('/events');
  es.onmessage=function(e){
    if(e.data==='reload'){ sessionStorage.setItem(k,scrollY); location.reload(); }
    else if(e.data.indexOf('error:')===0) show('build failed\\n\\n'+JSON.parse(e.data.slice(6)));
  };
})();
</script>`;

http.createServer((req,res)=>{
  if(req.url === '/events'){
    res.writeHead(200,{'Content-Type':'text/event-stream','Cache-Control':'no-cache','Connection':'keep-alive'});
    res.write('retry: 500\n\n');
    clients.add(res);
    req.on('close',()=>clients.delete(res));
    return;
  }
  let html;
  try{ html = fs.readFileSync(OUT,'utf8'); }
  catch{ res.writeHead(503); return res.end('not built yet'); }
  res.writeHead(200,{'Content-Type':'text/html; charset=utf-8','Cache-Control':'no-store'});
  res.end(html + LIVE);
}).listen(PORT, ()=>{
  console.log(`design system on http://localhost:${PORT}  (watching ${SRC})`);
  build();
});
