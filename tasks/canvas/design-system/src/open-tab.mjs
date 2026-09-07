// Writes /tmp/ds-tab.html with one tab of the artifact open, for screenshotting:
//
//   node open-tab.mjs docs
//   tasks/canvas/shot.sh --url file:///tmp/ds-tab.html --size 1400,2000
//
// Two things learned the hard way:
//
//   - APPEND the script. The generated document has no `</body>` tag, so every
//     injection anchored on one silently did nothing and the screenshot came
//     back showing the default tab (lessons.md #22).
//   - There is no scroll-to-heading option. The page's scrolling container is
//     not `window` — the sidebar is sticky and `main` scrolls — so scrolling
//     from outside lands somewhere blank. Capture from the top with a tall
//     `--size` instead.
import {readFileSync,writeFileSync} from 'fs';
const tab=process.argv[2]||'concepts';
const TABS=['concepts','scenarios','fixtures','docs'];
if(!TABS.includes(tab)){ console.error(`unknown tab "${tab}" — one of: ${TABS.join(', ')}`); process.exit(2); }
const src=readFileSync(new URL('../trace-design-system.html',import.meta.url).pathname,'utf8');
writeFileSync('/tmp/ds-tab.html', src+`
<script>
  document.querySelectorAll('.tab').forEach(function(t){
    t.classList.toggle('on', t.dataset.tab===${JSON.stringify(tab)});
  });
  ${JSON.stringify(TABS)}.forEach(function(id){
    var el=document.getElementById('t-'+id);
    if(el) el.hidden = (id!==${JSON.stringify(tab)});
  });
</script>`);
console.log('/tmp/ds-tab.html —', tab);
