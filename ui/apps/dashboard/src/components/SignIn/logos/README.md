# Sign-up trust panel — customer logos

Drop **white-mode single-colour SVGs** here, then register them in `index.ts`.
They are imported as React components via `vite-plugin-svgr`
(`import Replit from './replit.svg?react'`), the same pattern as
`src/components/Icons/*.svg?react`.

## Naming

Lowercase, kebab-case, no spaces or `=`: `replit.svg`, `elevenlabs.svg`,
`soundcloud.svg`. Figma exports arrive as `Customer=Name, Mode=White.svg` and
must be renamed — the original names are not importable.

## Why white-mode, not `currentColor`

The auth routes are pinned to a dark theme in `src/routes/__root.tsx`, so both
the gray panel and the form column behind these logos are dark at every
breakpoint. White fills are correct everywhere and no light variant is needed.

Do **not** bulk-replace `fill="white"` with `fill="currentColor"`. Some exports
use white fills _inside_ `<mask>` elements, where white means "reveal"; a
theme-dependent colour there would hide the artwork instead of colouring it.

## Two things every export needs fixed

Figma's output has caused a real bug on both counts, so check them:

**1. Strip `width` and `height` from the root `<svg>`.** Keep only `viewBox`.
The registry sets a per-logo height and the wall uses `w-auto`; a leftover
`width` attribute wins over the aspect ratio and renders the logo horizontally
squashed. This is silent — the logo still appears, just wrong.

**2. Flatten "solid path painted through a mask".** Figma sometimes emits the
letterforms as paths inside a `mask-type:alpha` mask and then paints one solid
path through it. That indirection does not survive inlining as JSX: the mask
stops applying and the solid path renders as a filled bar. `soundcloud.svg` had
this and was flattened to the letterform paths painted directly. A full-bounds
`mask-type:luminance` wrapper over real paths, as in `gitbook.svg`, is harmless
and can stay.

Also confirm: no `<style>` blocks or `class` attributes (svgr inlines them and
the names collide across files), no `<image>` payloads, and no duplicate `id`
values between files — svgr puts every logo in one document, so a shared
`clip0`/`mask0` id would let one logo clip another. Figma's `_49_xxx` suffixes
are usually unique enough.

## Sizing

Heights live per logo in `index.ts`, not as a shared class. These wordmarks
range from about 3:1 to 9.4:1, so a single height makes the widest read as
roughly twice the size of the narrowest; the values there balance them by
optical weight instead.

## Permissions

The eight in the approved set are cleared for use as endorsement. If that
changes for one, delete its file, its import, and its registry entry — nothing
else references them, and the wall reflows.
