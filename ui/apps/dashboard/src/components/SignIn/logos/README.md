# Sign-up trust panel — customer logos

Drop **single-color SVGs** here. These are imported as React components via
`vite-plugin-svgr` (`import Replit from './logos/replit.svg?react'`), which is
the same pattern used by `src/components/Icons/*.svg?react`.

## Files expected

One per logo, lowercase kebab-case, matching the approved mockup:

```
replit.svg
cubic.svg
elevenlabs.svg
cohere.svg
soundcloud.svg
gitbook.svg
resend.svg
avoca.svg
```

Order in the wall is set in code, not by filename — drop them in any order.
Fewer than eight is fine; the grid reflows. If a logo gets pulled for
permissions reasons, delete the file and remove its import.

## Why single-color

The wall renders on the dark mesh gradient on desktop **and** on
`canvasBase` in the left column on mobile — which is near-white in light mode.
One `currentColor` SVG recolors for both. Two-tone or full-color brand
lockups would need four files each and still look wrong on one of the two
backgrounds.

## SVG requirements

- **`viewBox` present.** Required — it's what makes the logo scale.
- **No hardcoded `width`/`height`** on the root `<svg>`, or they'll fight the
  CSS sizing. Strip them if your export tool adds them.
- **`fill="currentColor"`** on paths, or no `fill` attribute at all. Do **not**
  leave a hardcoded `fill="#000"` / `fill="#fff"` — it defeats the recoloring
  and the logo will vanish against one of the two backgrounds.
- **Flatten text to paths.** Wordmarks must not depend on a font being loaded.
- **No `<style>` blocks, no CSS classes.** svgr inlines these into the page
  and the class names collide across files.
- **No embedded raster** (`<image>` with a base64 payload) — it won't recolor
  and it defeats the point of using SVG.

Trimming the canvas to the artwork bounds helps; uneven padding inside the
viewBox makes optical alignment across eight logos much harder.

## Permissions

Sanjana confirmed these eight are cleared for use as endorsement. If that
changes for any single logo, deleting the file and its import is the whole
revert — nothing else references them.
