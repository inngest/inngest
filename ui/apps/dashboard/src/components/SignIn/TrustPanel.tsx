import LogoWall from './LogoWall';
import { customerLogos } from './logos';

/**
 * 1x1 transparent GIF, used as the `<img>` fallback so that below the `sm`
 * breakpoint -- where the whole panel is `display: none` -- no screenshot is
 * fetched. A plain `src` would still download on every phone, because
 * `display: none` does not cancel an image request.
 */
const BLANK_PIXEL =
  'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7';

const DESKTOP_ONLY = '(min-width: 640px)';

/**
 * Intrinsic size of the AVIF. Set as attributes so the browser reserves the
 * right box before the image arrives; the PNG fallback is a smaller export of
 * the same crop, so it shares this ratio.
 */
const SCREENSHOT_WIDTH = 1920;
const SCREENSHOT_HEIGHT = 989;

/**
 * Only a dark export exists, and only one is needed: the auth routes are
 * pinned to a dark theme in `__root.tsx`, so a light variant would never
 * render.
 */
function ProductScreenshot() {
  return (
    <picture>
      <source
        media={DESKTOP_ONLY}
        type="image/avif"
        srcSet="/images/auth/product-dark.avif"
      />
      <source
        media={DESKTOP_ONLY}
        type="image/png"
        srcSet="/images/auth/product-dark.png"
      />
      <img
        src={BLANK_PIXEL}
        alt=""
        width={SCREENSHOT_WIDTH}
        height={SCREENSHOT_HEIGHT}
        // Wider than its column on purpose, so it runs past the right edge of
        // the panel instead of shrinking to fit -- only the near portion of the
        // dashboard is meant to be visible. `max-w-none` is required to beat
        // the `max-width: 100%` the base stylesheet puts on images, and the
        // panel clips the overflow.
        className="border-subtle h-auto w-[150%] max-w-none rounded-lg border shadow-2xl"
      />
    </picture>
  );
}

export default function TrustPanel() {
  return (
    <div className="text-basis flex h-full flex-col items-center justify-center gap-14 overflow-hidden py-12">
      {/* Inset on the left only; the right side is the bleed. */}
      <div className="w-full pl-12">
        <ProductScreenshot />
      </div>

      {/* Gated together: the heading is meaningless without logos under it. */}
      {customerLogos.length > 0 && (
        <div className="flex flex-col items-center gap-6 px-8">
          <h2 className="font-mono text-xs uppercase tracking-widest opacity-60">
            Trusted by engineering teams at
          </h2>
          <LogoWall className="max-w-xl" />
        </div>
      )}
    </div>
  );
}
