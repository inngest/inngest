import LogoWall from '@/components/SignIn/LogoWall';
import SplitView from '@/components/SignIn/SplitView';
import TrustPanel from '@/components/SignIn/TrustPanel';
import { canonicalLink } from '@/utils/urls';
import { SignUp } from '@clerk/tanstack-react-start';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
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

export const Route = createFileRoute('/(auth)/sign-up/$')({
  component: RouteComponent,
  head: () => ({
    links: [canonicalLink('/sign-up')],
  }),
});

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
    <SplitView panel={<TrustPanel />}>
      <div className="my-auto flex w-full flex-col items-center gap-10">
        <div className="flex w-full flex-col items-center">
          {/* Rendered here rather than through Clerk's `logoImageUrl` so the
              mark comes from the SVG component, which inherits `currentColor`
              and needs no dark-mode filter. */}
          <InngestLogo className="text-basis mb-6" width={132} />

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

          {/* Trailing copy, tightest-to-loosest. The certification sits hard
              against the button it qualifies; the sign-in action and the legal
              fine print each get their own breathing room below it. */}
          <p className="text-basis mt-3 text-center font-mono text-[11px] uppercase tracking-wider">
            SOC 2 Type II certified &middot; Free tier, no card required
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

          <p className="text-subtle mt-5 max-w-xs text-center text-xs">
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
