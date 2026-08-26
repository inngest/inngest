# Sign-up trust panel — product screenshot

Drop **one dark PNG** named `product-dark.png`. The `.avif` beside it is
generated, not hand-authored.

Only a dark export is needed: the auth routes are pinned to a dark theme in
`src/routes/__root.tsx`, so a light variant would never render.

## Export spec

- **Width:** >= 1920px. The panel is 50vw and the image is rendered at 150% of
  that, so it is displayed wider than the panel and clipped.
- **Aspect:** flexible. The image is anchored left and bleeds off the right
  edge, so a wide screenshot loses its right portion rather than letterboxing.
  The current export is 1.94:1 and shows roughly its left two thirds.
- **Format:** PNG. Alpha is fine; it gets flattened during encoding.

## Before you export

Scrub anything real from the screenshot — this page is public and
unauthenticated:

- No real customer account names, org names, or user emails
- No real run IDs, event IDs, or signing keys that map to a live account
- No internal-only UI (feature-flagged panels, admin/impersonation chrome)

Synthetic data is fine and preferred.

## Regenerating the derived files

Use `sharp` from the pnpm store, **not** `sips`. `sips -s format avif` silently
zeroes the alpha channel on an RGBA source and writes a fully transparent file
that still reports correct dimensions — it looks like a working image until you
sample its pixels.

```sh
SHARP=ui/node_modules/.pnpm/sharp@0.34.5/node_modules/sharp
node -e "
const sharp=require('./$SHARP');
const A='ui/apps/dashboard/public/images/auth/';
sharp(A+'product-dark.png').resize(1920).removeAlpha().avif({quality:68,effort:6}).toFile(A+'product-dark.avif');
"
```

`removeAlpha()` is what keeps the transparency bug from reappearing.

The panel serves the AVIF first and falls back to the PNG for browsers without
AVIF decode (Safari below 16.4). Both are `media`-gated to `min-width: 640px`
so phones, where the panel is `display: none`, download neither.
