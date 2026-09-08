import LogoWall from '@/components/SignIn/LogoWall';
import SplitView from '@/components/SignIn/SplitView';
import TrustPanel from '@/components/SignIn/TrustPanel';
import { absoluteUrl, canonicalLink } from '@/utils/urls';
import { ClerkLoaded, ClerkLoading, SignUp } from '@clerk/tanstack-react-start';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { cn } from '@inngest/components/utils/classNames';
import { Link, createFileRoute, useLocation } from '@tanstack/react-router';

/**
 * Provisional marketing accent for the sign-up CTA, and only that button.
 *
 * This deliberately departs from `btnPrimary` (matcha green), which remains
 * the design system's primary button everywhere else. Marketing asked for the
 * inngest.com accent on this one surface while the colour's permanent role is
 * undecided, so it is scoped here rather than added to the shared token set --
 * deleting this constant and the `formButtonPrimary` override below reverts it
 * completely, with nothing to unpick from `globals.css`.
 *
 * `!` throughout because the provider styles the same button through the
 * `button` descriptor; Tailwind utilities share specificity, so source order
 * rather than class order would otherwise decide the winner.
 */
const MARKETING_CTA =
  '!bg-[#F65C4F] hover:!bg-[#DD5347] focus:!bg-[#DD5347] active:!bg-[#C54A3F]';

const getAnonymousId = () => {
  if (typeof document === 'undefined') {
    return null;
  }

  const cookie = document.cookie
    .split('; ')
    .find((c) => c.startsWith('ajs_anonymous_id='));

  return cookie ? cookie.split('=')[1] : null;
};

const TITLE = 'Create a free account | Inngest';
const DESCRIPTION =
  'Durable workflows, zero extra infra. Start on the free tier with no credit card required.';
/** Reuses the trust panel screenshot; at 1440x741 it is close to the 1.91:1 most cards expect. */
const SOCIAL_IMAGE = '/images/auth/product-dark.png';

export const Route = createFileRoute('/(auth)/sign-up/$')({
  component: RouteComponent,
  head: () => ({
    links: [canonicalLink('/sign-up')],
    // This URL is public and gets shared, so it needs its own title and a card
    // rather than inheriting the generic "Inngest Dashboard" from the root.
    meta: [
      { title: TITLE },
      { name: 'description', content: DESCRIPTION },
      { property: 'og:type', content: 'website' },
      { property: 'og:title', content: TITLE },
      { property: 'og:description', content: DESCRIPTION },
      { property: 'og:url', content: absoluteUrl('/sign-up') },
      { property: 'og:image', content: absoluteUrl(SOCIAL_IMAGE) },
      { property: 'og:image:width', content: '1440' },
      { property: 'og:image:height', content: '741' },
      { name: 'twitter:card', content: 'summary_large_image' },
      { name: 'twitter:title', content: TITLE },
      { name: 'twitter:description', content: DESCRIPTION },
      { name: 'twitter:image', content: absoluteUrl(SOCIAL_IMAGE) },
    ],
  }),
});

/**
 * Height reserved for Clerk's card so the column does not collapse and re-centre
 * when the form mounts client-side. Measured against the production field set
 * (email + password); an instance that also collects names renders ~83px taller
 * and will still grow past this, since `min-height` sets a floor, not a cap.
 *
 * Applied to the start step only. Deeper steps of the flow -- verification and
 * friends, which are reachable directly from an emailed link -- have their own
 * shapes and heights, and reserving this one for them would trade the shift it
 * removes here for a mismatch there.
 */
const FORM_MIN_HEIGHT = 'min-h-[508px]';

/**
 * Stand-in for Clerk's form while it mounts, so the reserved space is not a void.
 * Shaped after the start step (social buttons, divider, fields, submit), so it is
 * only rendered there.
 */
function FormSkeleton() {
  return (
    <div
      aria-hidden
      className="flex w-full max-w-[400px] animate-pulse flex-col gap-6 pt-14"
    >
      <div className="bg-canvasMuted mx-auto h-7 w-56 rounded" />
      <div className="flex flex-col gap-4">
        <div className="bg-canvasMuted h-10 rounded-md" />
        <div className="bg-canvasMuted h-10 rounded-md" />
      </div>
      <div className="bg-canvasMuted h-px w-full" />
      <div className="bg-canvasMuted h-10 rounded-md" />
      <div className="bg-canvasMuted h-10 rounded-md" />
      <div className="bg-canvasMuted h-10 rounded-md" />
    </div>
  );
}

function RouteComponent() {
  const anonymousId = getAnonymousId();
  const { pathname } = useLocation();

  // Clerk renders its "Already have an account?" action inside the card, which
  // puts it above our trust and legal copy. On the first step we hide that
  // footer and re-render the link below them instead, so the reading order is
  // CTA -> certification -> terms -> sign in. Later steps of the flow (email
  // verification and friends) live on deeper paths and keep Clerk's own
  // footer, which carries their back and alternate-method actions.
  const isStartStep = pathname === '/sign-up' || pathname === '/sign-up/';

  return (
    <SplitView panel={<TrustPanel />} panelBackground="neutral">
      <div className="my-auto flex w-full flex-col items-center gap-10">
        <div className="flex w-full flex-col items-center">
          {/* Rendered here rather than through Clerk's `logoImageUrl` so the
              mark comes from the SVG component, which inherits `currentColor`
              and needs no dark-mode filter. */}
          <InngestLogo className="text-basis mb-6" width={132} />

          {/* The wrapper stays in the tree on every step so that `SignUp` keeps
              its position and is not remounted when the step changes mid-flow;
              only the reservation and the placeholder are gated. */}
          <div
            className={cn(
              'flex w-full flex-col items-center',
              isStartStep && FORM_MIN_HEIGHT,
            )}
          >
            <ClerkLoading>{isStartStep ? <FormSkeleton /> : null}</ClerkLoading>
            <ClerkLoaded>
              <SignUp
                unsafeMetadata={{
                  ...(anonymousId && { anonymousID: anonymousId }),
                }}
                appearance={{
                  elements: {
                    footer: isStartStep ? 'hidden' : 'bg-none',
                    logoBox: 'hidden',
                    formButtonPrimary: MARKETING_CTA,
                    // Clerk's card carries 32px of bottom padding, which the
                    // trailing copy's own margin stacked on top of -- the
                    // certification line sat 44px below the button instead of the
                    // 12px intended. Dropped on the first step only; later steps
                    // still render Clerk's footer inside the card and need it.
                    card: isStartStep ? '!pb-0' : '',
                    // `!` because the provider sets `my-9` on the same descriptor;
                    // Tailwind utilities share specificity, so source order rather
                    // than class order would otherwise decide the winner.
                    header: '!mb-6 !mt-0',
                  },
                }}
              />
            </ClerkLoaded>
          </div>

          {/* Trailing copy, tightest-to-loosest. The certification sits hard
              against the button it qualifies; the sign-in action and the legal
              fine print each get their own breathing room below it. */}
          <p className="text-muted mt-3 text-center font-mono text-[11px] uppercase tracking-wider">
            SOC 2 Type II certified &middot;{' '}
            <span className="text-basis">Free tier, no card required</span>
          </p>

          {isStartStep && (
            <p className="text-basis mt-10 text-center text-sm">
              Already have an account?{' '}
              <Link
                to="/sign-in/$"
                params={{ _splat: '' }}
                className="text-link hover:underline"
              >
                Sign in
              </Link>
            </p>
          )}

          <p className="text-subtle mt-12 max-w-xs text-center text-xs">
            By signing up, you agree to our{' '}
            <a
              className="text-link hover:underline"
              href="https://inngest.com/terms"
              target="_blank"
              rel="noopener noreferrer"
            >
              Terms of Service
            </a>{' '}
            and{' '}
            <a
              className="text-link hover:underline"
              href="https://inngest.com/privacy"
              target="_blank"
              rel="noopener noreferrer"
            >
              Privacy Policy
            </a>
            .
          </p>
        </div>

        {/* The trust panel is hidden below `sm`, so the logo wall is repeated
            here to keep social proof on small screens. */}
        <LogoWall className="px-4 sm:hidden" />
      </div>
    </SplitView>
  );
}
