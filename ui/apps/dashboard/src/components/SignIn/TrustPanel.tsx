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
 *
 * To regenerate the AVIF from a new `product-dark.png`, use `sharp` from the
 * pnpm store -- not `sips`, which silently zeroes the alpha channel on an RGBA
 * source and writes a fully transparent file that still reports the right
 * dimensions:
 *
 *   node -e "require('<repo>/ui/node_modules/.pnpm/sharp@0.34.5/node_modules/sharp')\
 *     ('product-dark.png').resize(1920).removeAlpha()\
 *     .avif({quality:68,effort:6}).toFile('product-dark.avif')"
 *
 * `removeAlpha()` is what keeps that bug from reappearing.
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
        //
        // The multiplier sets where the crop lands, and is not arbitrary: the
        // visible fraction is its reciprocal, so 162% shows the leftmost ~62%
        // of the image. That falls on the panel divider just after the
        // Rerun/Cancel row, cutting before the event detail section rather
        // than through it. Widen it to crop earlier, narrow it to show more.
        // The left inset below only slides the image across; it cannot change
        // the crop, because the width is a percentage of the padded box and
        // both edges move together.
        className="border-subtle h-auto w-[162%] max-w-none rounded-lg border shadow-2xl"
      />
    </picture>
  );
}

export default function TrustPanel() {
  return (
    <div className="text-basis flex h-full flex-col justify-center gap-14 overflow-hidden py-12">
      {/* Takes its natural height and shrinks -- rather than growing -- so the
          screenshot is the only thing that gives way on a short panel while the
          group still centres on a tall one. Growing it instead pinned the image
          to the top and the logos to the bottom, leaving a gap between them.
          Its height scales with the panel's width -- it is rendered at 162% of
          it -- so on a wide, short panel it outgrows the available height. When
          it did that from a centred container, the excess spilled equally top
          and bottom and consumed the padding declared here, pushing the
          screenshot and the logo wall flush against the panel edges.

          Inset on the left only; the right side is the bleed. */}
      <div className="relative min-h-0 w-full overflow-hidden pl-16">
        <ProductScreenshot />
        {/* Softens the bottom edge so the vertical clip reads as the screenshot
            receding into the panel rather than a hard slice through the UI --
            it otherwise lands mid-row and cuts text in half.
            Painted in the panel's own colour, so it resolves itself: where the
            screenshot stops short of the bottom there is nothing under it but
            panel, and it disappears. Kept at 64px so it clears the slack on a
            1920x1080 panel, where the screenshot fits with ~75px to spare. */}
        <div
          aria-hidden
          className="from-canvasMuted pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t to-transparent"
        />
      </div>

      {/* Gated together: the heading is meaningless without logos under it. */}
      {customerLogos.length > 0 && (
        <div className="flex shrink-0 flex-col items-center gap-6 px-8">
          <h2 className="font-mono text-xs uppercase tracking-widest opacity-60">
            Trusted by engineering teams at
          </h2>
          <LogoWall className="max-w-2xl" />
        </div>
      )}
    </div>
  );
}
