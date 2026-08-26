import LogoWall from './LogoWall';
import { customerLogos } from './logos';

/**
 * Flip to `true` once `product-light.png` and `product-dark.png` land in
 * `public/images/auth` (see the README there). Rendering the `<picture>`
 * before the files exist would put broken-image icons on the sign-up page.
 */
const HAS_PRODUCT_SCREENSHOT = false;

/**
 * 1x1 transparent GIF, used as the `<img>` fallback so that below the `sm`
 * breakpoint -- where the whole panel is `display: none` -- no screenshot is
 * fetched. A plain `src` would still download on every phone, because
 * `display: none` does not cancel an image request.
 */
const BLANK_PIXEL =
  'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7';

const DESKTOP_ONLY = '(min-width: 640px)';

function ProductScreenshot({ theme }: { theme: 'light' | 'dark' }) {
  const base = `/images/auth/product-${theme}`;

  return (
    <picture>
      <source media={DESKTOP_ONLY} type="image/avif" srcSet={`${base}.avif`} />
      <source media={DESKTOP_ONLY} type="image/png" srcSet={`${base}.png`} />
      {/* TODO: set width/height from the real exports to pin the aspect ratio
          and avoid layout shift, once the assets land. */}
      <img
        src={BLANK_PIXEL}
        alt=""
        className="border-subtle w-full rounded-lg border shadow-2xl"
      />
    </picture>
  );
}

export default function TrustPanel() {
  return (
    // The mesh gradient is dark in both themes, so this column uses
    // `alwaysWhite` rather than theme-flipping foreground tokens.
    <div className="text-alwaysWhite flex h-full flex-col items-center justify-center gap-14 px-8 py-12">
      {HAS_PRODUCT_SCREENSHOT && (
        <div className="w-full max-w-2xl">
          <div className="dark:hidden">
            <ProductScreenshot theme="light" />
          </div>
          <div className="hidden dark:block">
            <ProductScreenshot theme="dark" />
          </div>
        </div>
      )}

      {/* Gated together: the heading is meaningless without logos under it. */}
      {customerLogos.length > 0 && (
        <div className="flex flex-col items-center gap-6">
          <h2 className="font-mono text-xs uppercase tracking-widest opacity-60">
            Trusted by engineering teams at
          </h2>
          <LogoWall className="max-w-xl" />
        </div>
      )}
    </div>
  );
}
