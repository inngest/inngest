# Sign-up trust panel — product screenshot

Drop **PNG only**. AVIF variants are generated from these; do not hand-author them.

## Files expected

| Filename            | Theme | Notes                               |
| ------------------- | ----- | ----------------------------------- |
| `product-light.png` | Light | Shown when the app is in light mode |
| `product-dark.png`  | Dark  | Shown when the app is in dark mode  |

Both are required. The panel picks one via the `dark:` class on `<html>`,
matching the existing `Lt-`/`Dk-` pairing in
`src/components/NavigationV2/Announcements/announcements.ts`.

## Export spec

- **Width:** >= 1440px (2x for a ~720px display width — the panel is 50vw on `md+`)
- **Aspect:** anywhere from 4:3 to 3:2. Content is top-anchored, so extra
  height crops from the bottom rather than the middle.
- **Format:** PNG, 8-bit. No transparency needed; the panel sits on the
  mesh gradient and the screenshot is inset with its own rounded corners.
- **Content:** runs list + trace timeline, per the approved mockup.

## Before you export

Scrub anything real from the screenshot — this page is public and unauthenticated:

- No real customer account names, org names, or user emails
- No real run IDs, event IDs, or signing keys that map to a live account
- No internal-only UI (feature-flagged panels, admin/impersonation chrome)

Synthetic data is fine and preferred.

## What happens after you drop them

AVIF variants are generated with `sips` and committed alongside the PNGs.
The panel renders a `<picture>` with an AVIF `<source>` and a PNG fallback,
gated on `(min-width: 640px)` so phones never download it.
