/**
 * Exports figures from the artifact as standalone SVGs for the user-facing docs.
 *
 * Two things have to be done to a figure before it can leave the page:
 *
 *   - the drawing resolves every colour through a CSS custom property defined
 *     on `:root`, so on its own it renders blank. The variables are inlined
 *     into each file.
 *   - the figures are built framed (minimap, Run row, finalization, blurred).
 *     That is a design-review device — in docs a blurred row reads as something
 *     being withheld — so the docs build runs with DS_FRAME=0.
 *
 * Run:  DS_FRAME=0 node ds.mjs && node export-docs.mjs && node ds.mjs
 */
import fs from 'fs';

const HERE=new URL('./',import.meta.url).pathname;
const OUT=HERE+'../../docs/images/';
const HTML=fs.readFileSync(HERE+'../trace-design-system.html','utf8');

// The variables every figure resolves its colours through.
//
// The value pattern is deliberately tight. It was `[^;]+`, which is greedy and
// happily crosses newlines — so an unterminated declaration in a style
// ATTRIBUTE (`style="--i:3;--s:0"`, no trailing semicolon) matched onward until
// the next `;` anywhere in the document, and every exported figure came out at
// 1.1MB with `</main>` and `<script>` inside it. The docs tab then inlined
// those and the page stopped parsing.
const VARS=[...new Set(HTML.match(/--[a-z0-9-]+: *[^;{}<>\n]{1,60};/g)||[])]
  .filter(v=>!/--(body|display|mono|figw|i|s):/.test(v))
  .join('');

const J=name=>JSON.parse(fs.readFileSync(HERE+name+'.json','utf8'));
const EX={...J('items-a'),...J('items-bc'),...J('items-disc'),...J('items-more')};

/** Which figure, which frame of it, and what the docs call the file. */
const WANT=[
  ['c0', 0,'sequential',        'Two sequential steps: queued, started, running, resolved.'],
  ['c1', 0,'fan-out',           'One discovery request reports three steps; the ribbon threads them.'],
  ['c2', 0,'fan-out-coalesce',  'Three steps resolve and cause the next request.'],
  ['c4', 0,'uneven-fan-out',    'An uneven fan-out, countable without interacting.'],
  ['w1', 0,'wait-matched',      'A wait that matched.'],
  ['w2', 0,'wait-timed-out',    'A wait that expired with no match.'],
  ['c25',0,'retry',             'Threw, backed off, retried, returned.'],
  ['c24',0,'caught-failure',    'A red step inside a green run.'],
  ['c27',0,'cancelled',         'Cancelled while executing.'],
  ['t1', 0,'compressed-axis',   'A seven-day idle stretch, compressed and marked.'],
  ['t2', 0,'compute-vs-elapsed','Seven days elapsed, 62ms executing.'],
  ['s1', 0,'collapsed-group',   'Five hundred steps collapsed to one row.'],
  ['s3', 0,'failure-cluster',   'A failure cluster, visible without interacting.'],
];

fs.mkdirSync(OUT,{recursive:true});
const manifest=[];
for(const [id,frame,name,alt] of WANT){
  const e=EX[id];
  if(!e){ console.error('missing figure',id); process.exitCode=1; continue; }
  const f=e.frames[frame];
  if(!f){ console.error('missing frame',id,frame); process.exitCode=1; continue; }
  // Inline the variables, and sit the drawing on the surface it was designed
  // against so it does not depend on the page behind it.
  const svg=f.svg.replace(
    /^<svg([^>]*)>/,
    (m,attrs)=>{
      // The background is sized to the viewBox, not to a huge arbitrary square.
      // The square was invisible while the SVG clipped to its viewport, and
      // became a page-covering slab the moment the artifact set
      // `overflow: visible` so figures could grow with the row pitch.
      const vb=(attrs.match(/viewBox="([^"]+)"/)||[])[1];
      const p=vb?vb.trim().split(/[ ,]+/).map(Number):null;
      const bg=(p&&p.length===4&&p.every(Number.isFinite))
        ? `<rect x="${p[0]}" y="${p[1]}" width="${p[2]}" height="${p[3]}" fill="var(--surface)"/>`
        : '';
      return `<svg${attrs} xmlns="http://www.w3.org/2000/svg">`+
        `<style>svg{${VARS}}</style>`+bg;
    });
  // A figure is a few KB. Anything near a megabyte means the harvest above has
  // swallowed the page again, and a silently enormous asset is exactly the kind
  // of thing that ships.
  if(svg.length > 60000){
    console.error(`${name}.svg is ${(svg.length/1024).toFixed(0)}KB — refusing to write it`);
    process.exitCode=1; continue;
  }
  fs.writeFileSync(OUT+name+'.svg',svg);
  manifest.push({id,name,alt,bytes:svg.length});
}
console.log(manifest.map(m=>`${m.name}.svg  ${String(m.bytes).padStart(6)}  (${m.id})`).join('\n'));
console.log('\n'+manifest.length+' figures ->',OUT);
