/**
 * Right-click a figure, "Copy image", paste it anywhere.
 *
 * A browser offers Copy image on an <img> and never on an inline <svg>, and the
 * figures have to stay inline: hover, selection and the row popover all live on
 * them. So one invisible <img> follows the figure under the pointer, holding a
 * PNG of exactly what that figure is drawing, and is put in the pointer's way
 * only for the right-click itself. The native menu finds an image under the
 * cursor; every other interaction still finds the figure.
 *
 * PNG rather than SVG because that is what a paste target reliably takes, and
 * because an SVG on its own would not look like the figure anyway: every colour
 * and row height is a CSS variable, some rules match on ancestors outside the
 * <svg>, and the fonts are the page's. None of that reaches an image. So each
 * element carries its own COMPUTED style into the copy -- the palette, the
 * spacing and the hovered row as they are on screen -- and the fonts are
 * embedded.
 *
 * Runs in the page only. Nothing here is used by the build.
 */

const NS = 'http://www.w3.org/2000/svg';

// Inherited, so written only where an element differs from its parent.
const INHERITED = ['fill', 'fill-opacity', 'fill-rule', 'stroke', 'stroke-width',
  'stroke-opacity', 'stroke-dasharray', 'stroke-dashoffset', 'stroke-linecap',
  'stroke-linejoin', 'paint-order', 'font-family', 'font-size', 'font-weight',
  'font-style', 'letter-spacing', 'text-anchor', 'dominant-baseline', 'visibility'];
// Not inherited, so written only where they differ from the initial value.
const OWN = { opacity: '1', filter: 'none', 'clip-path': 'none', mask: 'none',
  'mix-blend-mode': 'normal', transform: 'none' };
// Geometry the stylesheet sets -- bar heights, mark radii, the segment gap.
const GEOM = { rect: ['x', 'y', 'width', 'height', 'rx', 'ry'],
  circle: ['cx', 'cy', 'r'], ellipse: ['cx', 'cy', 'rx', 'ry'] };
// Never drawn themselves, so whatever `display` says, a reference needs them.
const DEFS = new Set(['defs', 'pattern', 'clipPath', 'mask', 'marker', 'filter',
  'linearGradient', 'radialGradient', 'symbol']);

// Computed paint servers can come back as absolute URLs, which point outside
// the image once it is on its own.
const localUrl = v => v.replace(/url\("?[^")]*#([^")]+)"?\)/g, 'url(#$1)');

/**
 * Copy the live element's computed style onto its clone, and its children's
 * onto theirs. Returns false where the element is not drawn, so the caller can
 * drop it rather than carry hidden variants into the image.
 */
function inline(live, copy, parent, inDefs, families) {
  const cs = getComputedStyle(live);
  inDefs = inDefs || DEFS.has(live.localName);
  if (cs.display === 'none' && !inDefs) return false;
  const out = [], mine = {};
  for (const p of INHERITED) {
    const v = cs.getPropertyValue(p);
    mine[p] = v;
    if (!parent || parent[p] !== v) out.push(p + ':' + localUrl(v));
  }
  for (const p in OWN) {
    const v = cs.getPropertyValue(p);
    if (v && v !== OWN[p]) out.push(p + ':' + localUrl(v));
  }
  if (cs.transform !== 'none')
    out.push('transform-origin:' + cs.transformOrigin, 'transform-box:' + cs.transformBox);
  for (const p of GEOM[live.localName] || []) {
    const v = cs.getPropertyValue(p);
    if (v && v !== 'auto') out.push(p + ':' + v);
  }
  if (live.localName === 'text' || live.localName === 'tspan')
    families.add(mine['font-family'].split(',')[0].trim().replace(/^["']|["']$/g, ''));
  // What the attributes said is already in the computed style, resolved. Left
  // in, a `var()` would resolve to nothing in the image and override it.
  for (const a of [...copy.attributes])
    if (a.name === 'style' || a.name.startsWith('data-') || a.value.includes('var('))
      copy.removeAttribute(a.name);
  if (out.length) copy.setAttribute('style', out.join(';'));
  const lk = live.children, ck = copy.children;
  for (let i = ck.length - 1; i >= 0; i--)
    if (!inline(lk[i], ck[i], mine, inDefs, families)) ck[i].remove();
  return true;
}

/**
 * A figure can point at a pattern it does not define -- the hatch lives in the
 * panel's hidden svg as well as in the figures. Anything referenced and missing
 * is borrowed from the page, styled the same way.
 */
function borrowDefs(copy, families) {
  const text = new XMLSerializer().serializeToString(copy);
  const ids = new Set([...text.matchAll(/url\(#([^)"]+)\)|href="#([^"]+)"/g)].map(m => m[1] || m[2]));
  let defs = null;
  for (const id of ids) {
    if (copy.querySelector('#' + CSS.escape(id))) continue;
    const live = document.getElementById(id);
    if (!live) continue;
    const c = live.cloneNode(true);
    inline(live, c, null, true, families);
    if (!defs) { defs = document.createElementNS(NS, 'defs'); copy.insertBefore(defs, copy.firstChild); }
    defs.appendChild(c);
  }
}

/**
 * The page's own web fonts, as data URLs an image can use.
 *
 * Fetched from the stylesheet the page already links, once, and only the Latin
 * subsets: that is every glyph a figure draws, and the rest would multiply the
 * size of every copy. Offline it quietly gives nothing, and the copy falls
 * back to the system's fonts rather than failing.
 */
let FACES = null;
const DATA = new Map();
function faces() {
  if (!FACES) FACES = (async () => {
    const link = [...document.querySelectorAll('link[rel="stylesheet"]')]
      .find(l => l.href.includes('fonts.googleapis.com'));
    if (!link) return [];
    const css = await (await fetch(link.href)).text();
    const out = [];
    for (const m of css.matchAll(/\/\*\s*([a-z-]+)\s*\*\/\s*(@font-face\s*\{[^}]*\})/g)) {
      if (m[1] !== 'latin' && m[1] !== 'latin-ext') continue;
      const fam = (m[2].match(/font-family:\s*'([^']+)'/) || [])[1];
      const url = (m[2].match(/url\((https:[^)]+)\)/) || [])[1];
      if (fam && url) out.push({ fam, url, block: m[2] });
    }
    return out;
  })().catch(() => []);
  return FACES;
}
function dataUrl(url) {
  if (!DATA.has(url)) DATA.set(url, fetch(url).then(r => r.blob()).then(b =>
    new Promise((ok, no) => { const f = new FileReader();
      f.onload = () => ok(f.result); f.onerror = no; f.readAsDataURL(b); })));
  return DATA.get(url);
}
async function fontCSS(families) {
  const list = (await faces()).filter(f => families.has(f.fam));
  const blocks = await Promise.all(list.map(async f => {
    try { return f.block.replace(f.url, await dataUrl(f.url)); } catch { return ''; }
  }));
  return blocks.join('');
}

/** The figure as a PNG: at least twice its size on screen, on its own card. */
async function png(svg) {
  const r = svg.getBoundingClientRect();
  if (!r.width || !r.height) return null;
  const copy = svg.cloneNode(true), families = new Set();
  inline(svg, copy, null, false, families);
  borrowDefs(copy, families);
  const scale = Math.max(2, devicePixelRatio || 1), pad = 12;
  copy.setAttribute('xmlns', NS);
  copy.setAttribute('width', String(r.width * scale));
  copy.setAttribute('height', String(r.height * scale));
  const css = await fontCSS(families);
  if (css) {
    const st = document.createElementNS(NS, 'style');
    st.textContent = css;
    copy.insertBefore(st, copy.firstChild);
  }
  const src = URL.createObjectURL(new Blob([new XMLSerializer().serializeToString(copy)],
    { type: 'image/svg+xml' }));
  try {
    const im = new Image();
    im.src = src;
    await im.decode();
    const c = document.createElement('canvas');
    c.width = Math.round((r.width + pad * 2) * scale);
    c.height = Math.round((r.height + pad * 2) * scale);
    const g = c.getContext('2d');
    g.fillStyle = getComputedStyle(svg.closest('.figure') || document.body).backgroundColor;
    g.fillRect(0, 0, c.width, c.height);
    g.drawImage(im, pad * scale, pad * scale, r.width * scale, r.height * scale);
    return c.toDataURL('image/png');
  } finally {
    URL.revokeObjectURL(src);
  }
}

// ---- the overlay ---------------------------------------------------------

const img = document.createElement('img');
img.alt = '';
img.setAttribute('aria-hidden', 'true');
img.className = 'figcopy';
Object.assign(img.style, { position: 'fixed', left: '0', top: '0', width: '0', height: '0',
  opacity: '0', pointerEvents: 'none', zIndex: '40' });
document.body.appendChild(img);

let ready = null;   // the figure the image currently holds
let serial = 0, buildTimer = 0, armTimer = 0;
let armed = 0;      // 1: right button down over a figure, 2: its menu is open

const figureOf = t => {
  const f = t && t.closest && t.closest('main .figure');
  return f && f.querySelector('svg');
};

/**
 * Rebuilt whenever the pointer comes to rest on a figure.
 *
 * Built ahead of the click because the menu does not wait: on some platforms it
 * opens on the same press. And rebuilt on every rest, not once per figure, so
 * the copy is what is on screen -- a hovered row, a toggled feature, a palette
 * change -- rather than the figure as it was first seen.
 */
document.addEventListener('mouseover', e => {
  if (armed) return;
  const svg = figureOf(e.target);
  if (!svg) return;
  clearTimeout(buildTimer);
  buildTimer = setTimeout(async () => {
    const n = ++serial;
    let url = null;
    try { url = await png(svg); } catch (err) { console.warn('figure copy:', err); }
    if (n !== serial || !url) return;   // superseded, or it failed
    img.src = url;
    ready = svg;
  }, 120);
});

function arm(svg) {
  const r = svg.getBoundingClientRect();
  Object.assign(img.style, { left: r.left + 'px', top: r.top + 'px',
    width: r.width + 'px', height: r.height + 'px', pointerEvents: 'auto' });
  armed = 1;
  clearTimeout(armTimer);
  // A press that never became a menu must not leave the figure covered.
  armTimer = setTimeout(() => { if (armed === 1) disarm(); }, 1500);
}
function disarm() {
  armed = 0;
  Object.assign(img.style, { pointerEvents: 'none', width: '0', height: '0' });
}

// Capture phase, so the image is in place before the menu looks for a target.
document.addEventListener('pointerdown', e => {
  if (e.target === img && e.button === 2) return;   // still over the figure it covers
  disarm();
  if (e.button !== 2 || e.pointerType !== 'mouse') return;
  const svg = figureOf(e.target);
  if (svg && svg === ready && img.complete && img.naturalWidth) arm(svg);
}, true);
document.addEventListener('contextmenu', e => {
  if (e.target === img) armed = 2; else if (armed) disarm();
}, true);
// Once the menu has closed, the first movement hands the figure back.
document.addEventListener('pointermove', () => { if (armed === 2) disarm(); }, true);
document.addEventListener('wheel', () => { if (armed) disarm(); }, { capture: true, passive: true });
