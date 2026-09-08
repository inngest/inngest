/**
 * The source of each fixture's function, read from the file that defines it.
 *
 * The shapes are written once, in `canvas_pairs_v4.ts`, and run on both
 * clients. The page shows the same text -- extracted here rather than
 * transcribed -- so a shape cannot be edited without the page following, which
 * is the mistake every hand-copied snippet in this artifact has eventually
 * made.
 */
import fs from 'fs';

const SRC=new URL('../../../../tests/js/src/inngest/canvas_pairs_v4.ts',
  import.meta.url).pathname;

/** The body of `handler:` for each shape, keyed by its id. */
export function fixtureCode(){
  let text;
  try{ text=fs.readFileSync(SRC,'utf8'); }catch{ return {}; }
  const out={};
  // One segment per shape, so a handler written as an expression -- `async () =>
  // "done"` -- does not let the search run on and swallow the next shape's id.
  const ids=[...text.matchAll(/id:\s*"([^"]+)"/g)];
  for(let k=0;k<ids.length;k++){
    const seg=text.slice(ids[k].index, k+1<ids.length?ids[k+1].index:text.length);
    // `handler:` for a shape in the list; a bare arrow for the few built
    // alongside it, which need something in scope the list cannot give them.
    const h=seg.match(/handler:\s*async\s*\([^)]*\)\s*=>\s*/)
      || seg.match(/,\s*async\s*\([^)]*\)\s*=>\s*/);
    if(!h) continue;
    const rest=seg.slice(h.index+h[0].length);
    if(rest[0]==='{'){
      let i=1, depth=1;
      while(i<rest.length && depth>0){
        const c=rest[i];
        if(c==='{') depth++;
        else if(c==='}') depth--;
        else if(c==='"'||c==="'"||c==='`'){            // skip strings whole
          const q=c; i++;
          while(i<rest.length && rest[i]!==q){ if(rest[i]==='\\') i++; i++; }
        }
        i++;
      }
      out[ids[k][1].replace(/^pair-/,'')]=dedent(rest.slice(1, i-1));
    }else{
      // An expression body is the whole function: show it as a return.
      const line=rest.split('\n')[0].replace(/[\s,;}]+$/,'');
      out[ids[k][1].replace(/^pair-/,'')]='return '+line+';';
    }
  }
  return out;
}

/** Strip the common indent and the blank lines at either end. */
function dedent(s){
  const lines=s.replace(/\t/g,'  ').split('\n');
  while(lines.length && !lines[0].trim()) lines.shift();
  while(lines.length && !lines[lines.length-1].trim()) lines.pop();
  const pad=Math.min(...lines.filter(l=>l.trim()).map(l=>l.match(/^ */)[0].length));
  return lines.map(l=>l.slice(pad)).join('\n');
}
