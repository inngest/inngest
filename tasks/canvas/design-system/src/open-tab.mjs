// Writes /tmp/ds-tab.html with one tab of the artifact open, for screenshotting:
//
//   node open-tab.mjs docs
//   tasks/canvas/shot.sh --url file:///tmp/ds-tab.html --size 1400,2200
//
// Three things learned the hard way:
//
//   - APPEND the script. The generated document has no `</body>` tag, so an
//     injection anchored on one silently did nothing and the screenshot came
//     back showing the default tab (lessons.md #22).
//   - CLICK the real button. Setting `hidden` directly bypasses the page's own
//     handler — which is exactly how a broken handler shipped: the screenshot
//     looked right and the served page was empty.
//   - No scroll-to-heading. The page's scrolling container is not `window`, so
//     scrolling from outside lands somewhere blank. Capture from the top with a
//     tall `--size`.
import {readFileSync,writeFileSync} from 'fs';

const tab=process.argv[2]||'concepts';
const src=readFileSync(new URL('../trace-design-system.html',import.meta.url).pathname,'utf8');

const known=[...src.matchAll(/data-tab="([a-z]+)"/g)].map(m=>m[1]);
if(!known.includes(tab)){
  console.error(`unknown tab "${tab}" — the page has: ${known.join(', ')}`);
  process.exit(2);
}

writeFileSync('/tmp/ds-tab.html', src+`
<script>
  var btn = document.querySelector('.tab[data-tab=' + ${JSON.stringify(JSON.stringify(tab))} + ']');
  if (!btn) throw new Error('no tab button for ${tab}');
  btn.click();
  var shown = document.querySelector('main section:not([hidden])');
  if (!shown) throw new Error('clicking ${tab} left every section hidden');
</script>`);
console.log('/tmp/ds-tab.html —', tab, `(page has: ${known.join(', ')})`);
